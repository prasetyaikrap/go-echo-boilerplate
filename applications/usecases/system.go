package usecases

import "go-serviceboilerplate/infrastrucutres/repositories"

type SystemUsecase struct {
	systemRepository *repositories.SystemRepositories
}

func NewSystemUsecase(systemRepository *repositories.SystemRepositories) *SystemUsecase {
	return &SystemUsecase{systemRepository}
}

func (r *SystemUsecase) GetSystemInfo() map[string]string {
	data := r.systemRepository.GetSystemInfo()

	return data
}