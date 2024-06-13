package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	"github.com/kardianos/service"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

type NginxConfigService struct {
	ConfigDir  string
	NginxPath  string
	backupFile string
	Lock       sync.Mutex
	Logger     *logrus.Logger
}

type ConfigFile struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

func NewNginxConfigService(configDir, nginxPath, logDir string) *NginxConfigService {
	if logDir != "" {
		_, err := os.Stat(logDir)
		if os.IsNotExist(err) {
			err := os.MkdirAll(logDir, 0755) // Create directory with permission 0755 (rwxr-xr-x)
			if err != nil {
				log.Fatalf("Error creating log directory: %v", err)
			}
		}
		logger := logrus.New()
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
		logger.SetOutput(&lumberjack.Logger{
			Filename:   filepath.Join(logDir, "nginx-agent.log"),
			MaxSize:    10, // 10 MB per file
			MaxBackups: 5,  // 5 backups
			MaxAge:     30, // 30 days
		})
		return &NginxConfigService{
			ConfigDir: configDir,
			NginxPath: nginxPath,
			Logger:    logger,
		}
	}

	return &NginxConfigService{
		ConfigDir: configDir,
		NginxPath: nginxPath,
	}
}

func (s *NginxConfigService) logMessage(msg string) {
	if s.Logger != nil {
		s.Logger.Info(msg)
	} else {
		log.Println(msg)
	}
}

func (s *NginxConfigService) backupConfig(filename string) error {
	backupFile := filepath.Join(s.ConfigDir, filename+".bak")
	originalFile := filepath.Join(s.ConfigDir, filename)
	if _, err := os.Stat(originalFile); err == nil {
		err := copyFile(backupFile, originalFile)
		if err != nil {
			s.logMessage("Error backing up config file " + filename + ": " + err.Error())
			return err
		}
		s.logMessage("Config file " + filename + " backed up successfully")
	}
	return nil
}

func (s *NginxConfigService) restoreConfig(filename string) error {
	if s.backupFile != "" {
		originalFile := filepath.Join(s.ConfigDir, filename)
		err := copyFile(originalFile, s.backupFile)
		if err != nil {
			s.logMessage("Error restoring config file " + filename + ": " + err.Error())
			return err
		}
		err = s.clearBackup()
		if err != nil {
			s.logMessage("Error clearing backup file: " + err.Error())
			return err
		}
		s.logMessage("Config file " + filename + " restored successfully")
	}
	return nil
}

func (s *NginxConfigService) clearBackup() error {
	if s.backupFile != "" {
		err := os.Remove(s.backupFile)
		if err != nil {
			s.logMessage("Error clearing backup file: " + err.Error())
			return err
		}
		s.logMessage("Backup file cleared successfully")
		s.backupFile = ""
	}
	return nil
}
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (s *NginxConfigService) reloadNginx() error {
	s.logMessage("Reloading Nginx...")
	cmd := exec.Command(s.NginxPath, "-s", "reload")
	return cmd.Run()
}

func (s *NginxConfigService) UploadConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	filename := vars["filename"]
	if filename == "" {
		http.Error(w, "Missing filename in request", http.StatusBadRequest)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		http.Error(w, "Missing content in request", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(s.ConfigDir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	s.Lock.Lock()
	defer s.Lock.Unlock()
	_ = s.backupConfig(filename) // Backup config before updating

	err = os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		_ = s.restoreConfig(filename) // Rollback if write fails
		return
	}
	s.logMessage("upload file:" + filename + "\n" + content)

	if err := s.reloadNginx(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		_ = s.restoreConfig(filename) // Rollback if restart fails
		return
	}
	s.logMessage("Config file " + filename + " uploaded successfully")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Config uploaded successfully"))
}
func (s *NginxConfigService) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	filename := vars["filename"]
	if filename == "" {
		http.Error(w, "Missing filename in request", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(s.ConfigDir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	s.Lock.Lock()
	defer s.Lock.Unlock()
	_ = s.backupConfig(filename) // Backup config before deleting
	// 读取原文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		_ = s.restoreConfig(filename) // Rollback if deletion fails
		return
	}
	s.logMessage("delete file:" + filename + "\n" + string(content))
	err = os.Remove(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		_ = s.restoreConfig(filename) // Rollback if deletion fails
		return
	}

	if err := s.reloadNginx(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		_ = s.restoreConfig(filename) // Rollback if restart fails
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Config deleted successfully"))
}

func (s *NginxConfigService) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	filename := vars["filename"]
	if filename == "" {
		http.Error(w, "Missing filename in request", http.StatusBadRequest)
		return
	}
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		http.Error(w, "Missing content in request", http.StatusBadRequest)
		return
	}
	filePath := filepath.Join(s.ConfigDir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	s.Lock.Lock()
	defer s.Lock.Unlock()
	_ = s.backupConfig(filename) // Backup config before updating
	// 读取原文件内容
	old_content, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		_ = s.restoreConfig(filename) // Rollback if write fails
		return
	}
	s.logMessage("update file:" + filename + "\nfrom:\n" + string(old_content) + "\n" + content + "\nto:\n" + content)
	err = os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		_ = s.restoreConfig(filename) // Rollback if write fails
		return
	}

	if err := s.reloadNginx(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		_ = s.restoreConfig(filename) // Rollback if restart fails
		return
	}
	s.logMessage("Config file " + filename + " updated successfully")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Config updated successfully"))
}

func (s *NginxConfigService) ListConfigs(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(s.ConfigDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var configFiles []ConfigFile
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if filepath.Ext(file.Name()) == ".conf" {
			filePath := filepath.Join(s.ConfigDir, file.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			configFiles = append(configFiles, ConfigFile{
				Filename: file.Name(),
				Content:  string(content),
			})
		}
	}

	response, err := json.Marshal(configFiles)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.logMessage("List configs:" + strconv.Itoa(len(configFiles)) + "successfully")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response)
}

type program struct {
	server *http.Server
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) Stop(s service.Service) error {
	if p.server != nil {
		return p.server.Shutdown(nil)
	}
	return nil
}

func getConfigDir() string {
	configDir := os.Getenv("NGX_CONF_API_CONFIG_DIR")
	if configDir != "" {
		return configDir
	}
	return "/etc/nginx/conf.d/"
}

func getNginxPath() string {
	nginxPath := os.Getenv("NGX_CONF_API_NGINX_PATH")
	if nginxPath != "" {
		return nginxPath
	}
	return "/usr/sbin/nginx"
}

func getLogDir() string {
	logDir := os.Getenv("NGX_CONF_API_LOG_DIR")
	if logDir != "" {
		return logDir
	}
	return "/var/log/nginx/agent"
}

func getHost() string {
	host := os.Getenv("NGX_CONF_API_HOST")
	if host != "" {
		return host
	}
	return "0.0.0.0"
}

func getPort() int {
	portStr := os.Getenv("NGX_CONF_API_PORT")
	if portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			log.Fatalf("Invalid port: %s", portStr)
		}
		return port
	}
	return 5000
}

func (p *program) run() {
	var configDir, nginxPath, logDir string
	var host string
	var port int
	flag.StringVar(&configDir, "configDir", getConfigDir(), "Nginx configuration directory")
	flag.StringVar(&nginxPath, "nginxPath", getNginxPath(), "Path to Nginx executable")
	flag.StringVar(&logDir, "logDir", getLogDir(), "Log directory")
	flag.IntVar(&port, "port", getPort(), "Port to listen on")
	flag.StringVar(&host, "host", getHost(), "Host to listen on")
	flag.Parse()

	nginxService := NewNginxConfigService(configDir, nginxPath, logDir)

	r := mux.NewRouter()
	api := r.PathPrefix("/api/ngx/").Subrouter()

	api.HandleFunc("/configs/{filename}", nginxService.UploadConfig).Methods("POST")
	api.HandleFunc("/configs/{filename}", nginxService.DeleteConfig).Methods("DELETE")
	api.HandleFunc("/configs/{filename}", nginxService.UpdateConfig).Methods("PUT")
	api.HandleFunc("/configs", nginxService.ListConfigs).Methods("GET")

	addr := host + ":" + strconv.Itoa(port)
	log.Printf("Starting Nginx Config Service on %s...", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func main() {
	svcConfig := &service.Config{
		Name:        "NginxConfigService",
		DisplayName: "Nginx Config Service",
		Description: "Service for managing Nginx configurations dynamically.",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	if err = s.Run(); err != nil {
		log.Fatal(err)
	}
}
