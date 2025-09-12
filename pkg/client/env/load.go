package env

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

type ServiceAccount struct {
	ProjectName  string
	ClientID     string
	ClientSecret string
}

func (s *ServiceAccount) String() string {
	return fmt.Sprintf(
		"ServiceAccount{ProjectName: %s, ClientID: %s, ClientSecret: %s}",
		s.ProjectName, s.ClientID, s.ClientSecret,
	)
}

func (s *ServiceAccount) GetProjectName() string {
	return s.ProjectName
}

func Load() *ServiceAccount {

	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	godotenv.Load(filepath.Join(dir, ".env"))

	return &ServiceAccount{
		ProjectName:  os.Getenv("PROJECT_NAME"),
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
	}
}
