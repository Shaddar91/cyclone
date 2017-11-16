/*
Copyright 2017 caicloud authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package conversion

import (
	"fmt"
	"strings"
	"time"

	"github.com/caicloud/cyclone/api"
	newapi "github.com/caicloud/cyclone/pkg/api"
	"gopkg.in/yaml.v2"
)

const (
	// AdminUserID represents the default user id of service.
	AdminUserID string = "admin"

	// AdminUsername represents the default username of service.
	AdminUsername string = "admin"
)

// ConvertPipelineToService converts the pipeline to service, as the running of pipeline dependents on service. This
// is just a workaround, the pipeline will can be run directly in the future.
func ConvertPipelineToService(projectName string, pipeline *newapi.Pipeline) (*api.Service, error) {
	service := &api.Service{}

	// Basic information of service.
	service.Name = pipeline.Name
	service.Description = pipeline.Description

	if pipeline.Build == nil && pipeline.Build.Stages == nil && pipeline.Build.Stages.CodeCheckout == nil {
		return nil, fmt.Errorf("fail to generate service as code checkout stages is empty")
	}

	// Convert the repository for the service.
	repository, err := convertRepository(pipeline.Build.Stages.CodeCheckout)
	if err != nil {
		return nil, err
	}
	service.Repository = *repository

	service.UserID = AdminUserID
	// Username is used as the repo of built image.
	service.Username = projectName

	// Convert the build stages to caicloud.yml string for service.
	caicloudYamlStr, err := convertBuildStagesToCaicloudYaml(pipeline)
	if err != nil {
		return nil, err
	}
	service.CaicloudYaml = caicloudYamlStr

	// Have checked the correction of buildInfos in function convertBuildStagesToCaicloudYaml().
	if pipeline.Build.Stages.ImageBuild != nil {
		service.Dockerfile = pipeline.Build.Stages.ImageBuild.BuildInfos[0].Dockerfile
		service.ImageName = pipeline.Build.Stages.ImageBuild.BuildInfos[0].ImageName
	}

	return service, nil
}

// convertBuildStagesToCaicloudYaml converts the config of build stages in pipeline to caicloud.yml.
func convertBuildStagesToCaicloudYaml(pipeline *newapi.Pipeline) (string, error) {
	if pipeline.Build == nil || pipeline.Build.BuilderImage == nil || pipeline.Build.Stages == nil {
		return "", fmt.Errorf("fail to generate caicloud.yml as builder image or build stages is empty")
	}

	builderImage := pipeline.Build.BuilderImage
	stages := pipeline.Build.Stages
	if stages.Package == nil {
		return "", fmt.Errorf("fail to generate caicloud.yml as package stages is empty")
	}

	caicloudYAMLConfig := &Config{}

	// Convert the package stage of pipeline to the prebuild of caicloud.yml.
	preBuild := &PreBuild{}
	packageConfig := stages.Package
	preBuild.Commands = packageConfig.Command
	preBuild.Outputs = packageConfig.Outputs
	preBuild.Image = builderImage.Image
	preBuild.Environment = convertEnvVars(builderImage.EnvVars)

	caicloudYAMLConfig.PreBuild = preBuild

	if stages.ImageBuild != nil {
		// Convert the image build stage to the build of caicloud.yml.
		build := &Build{}
		imageBuildConfig := stages.ImageBuild
		buildInfoNum := len(imageBuildConfig.BuildInfos)
		if buildInfoNum == 0 || buildInfoNum > 1 {
			return "", fmt.Errorf("fail to generate caicloud.yml as %d buildInfos provided in imageBuild stage", buildInfoNum)
		}

		// Now only support one build info.
		buildInfo := imageBuildConfig.BuildInfos[0]
		build.ContextDir = buildInfo.ContextDir
		build.DockerfileName = buildInfo.DockerfilePath

		caicloudYAMLConfig.Build = build
	}

	// Convert the integration test stage of pipeline to the integration of caicloud.yml.
	if stages.IntegrationTest != nil {
		integration := &Integration{}
		integrationTestConfig := stages.IntegrationTest
		services := make(map[string]Service)
		for _, service := range integrationTestConfig.Services {
			services[service.Name] = Service{
				Image:       service.Image,
				Environment: convertEnvVars(service.EnvVars),
				Commands:    service.Command,
			}
		}

		integration.Services = services
		integration.Commands = integrationTestConfig.Config.Command

		caicloudYAMLConfig.Integration = integration
	}

	config, err := yaml.Marshal(caicloudYAMLConfig)
	if err != nil {
		return "", err
	}

	return string(config), nil
}

// convertEnvVars converts the environment variables to the string list for caicloud.yml.
func convertEnvVars(envVars []newapi.EnvVar) []string {
	environment := []string{}
	for _, envVar := range envVars {
		environment = append(environment, fmt.Sprintf("%s=%s", envVar.Name, envVar.Value))
	}

	return environment
}

// convertRepository converts the code checkout stage config of pipeline to the repository config of service.
func convertRepository(codeCheckoutStage *newapi.CodeCheckoutStage) (*api.ServiceRepository, error) {
	if codeCheckoutStage == nil {
		return nil, fmt.Errorf("fail to generate service repository as code checkout stages is empty")
	}

	codeSourceNumber := len(codeCheckoutStage.CodeSources)
	if codeSourceNumber == 0 || codeSourceNumber > 1 {
		return nil, fmt.Errorf("only support one code source, but get %d", codeSourceNumber)
	}

	codeSource := codeCheckoutStage.CodeSources[0]
	serviceRepository := &api.ServiceRepository{}
	switch codeSource.Type {
	case newapi.GitLab:
		gitSource := codeSource.GitLab
		serviceRepository.Vcs = api.Git
		serviceRepository.SubVcs = api.GITLAB
		serviceRepository.Webhook = api.GITLAB
		serviceRepository.URL = gitSource.Url
		serviceRepository.Username = gitSource.Username
		serviceRepository.Password = gitSource.Password
	case newapi.GitHub:
		gitSource := codeSource.GitHub
		serviceRepository.Vcs = api.Git
		serviceRepository.SubVcs = api.GITHUB
		serviceRepository.Webhook = api.GITHUB
		serviceRepository.URL = gitSource.Url
		serviceRepository.Username = gitSource.Username
		serviceRepository.Password = gitSource.Password
	case newapi.SVN:
		gitSource := codeSource.GitHub
		serviceRepository.Vcs = api.Svn
		serviceRepository.URL = gitSource.Url
		serviceRepository.Username = gitSource.Username
		serviceRepository.Password = gitSource.Password
	default:
		return nil, fmt.Errorf("not support the code source type %s", codeSource.Type)
	}

	return serviceRepository, nil
}

// ConvertPipelineParamsToVersion converts the pipeline perform params to run the pipeline.
func ConvertPipelineParamsToVersion(performParams *newapi.PipelinePerformParams) *api.Version {
	version := &api.Version{
		Description:   performParams.Description,
		Status:        api.VersionPending,
		SecurityCheck: false,
		CreateTime:    time.Now(),
	}

	stagesStr := strings.Join(performParams.Stages, ",")
	version.Operation = api.VersionOperation(strings.Replace(stagesStr, "imageRelease", "publish", 1))

	version.Name = performParams.Name

	if performParams.CreateSCMTag {
		version.Operator = api.APIOperator
	}

	return version
}

// ConvertVersionToPipelineRecord converts the version to pipeline record.
func ConvertVersionToPipelineRecord(version *api.Version) (*newapi.PipelineRecord, error) {
	pipelineRecord := &newapi.PipelineRecord{
		ID:         version.VersionID,
		PipelineID: version.ServiceID,
		VersionID:  version.VersionID,
		Name:       version.Name,
		StartTime:  version.CreateTime,
		EndTime:    version.EndTime,
	}

	status, err := convertVersionStatusToPipelineRecordStatus(version.Status)
	if err != nil {
		return nil, fmt.Errorf("fail to convert version status as %s", err.Error())
	}
	pipelineRecord.Status = status

	// Convert the performed stages.
	performStagesStr := strings.Replace(string(version.Operation), "publish", "imageRelease", 1)
	performParams := &newapi.PipelinePerformParams{
		Stages: strings.Split(performStagesStr, ","),
	}
	pipelineRecord.PerformParams = performParams

	return pipelineRecord, nil
}

// convertVersionStatusToPipelineRecordStatus converts the version status to pipeline record status.
func convertVersionStatusToPipelineRecordStatus(status api.VersionStatus) (newapi.Status, error) {
	convertionMap := map[api.VersionStatus]newapi.Status{
		api.VersionPending: newapi.Pending,
		api.VersionRunning: newapi.Running,
		api.VersionHealthy: newapi.Success,
		api.VersionFailed:  newapi.Failed,
		api.VersionCancel:  newapi.Aborted,
	}

	if value, ok := convertionMap[status]; ok {
		return value, nil
	}

	return newapi.Status("Unknown"), fmt.Errorf("The version status %s is not supported", status)
}