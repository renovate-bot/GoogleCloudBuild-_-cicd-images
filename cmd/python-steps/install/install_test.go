// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package install

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/GoogleCloudBuild/cicd-images/cmd/python-steps/internal/auth"
	"github.com/GoogleCloudBuild/cicd-images/cmd/python-steps/internal/command"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

var mockLogger = zap.NewNop()

func setupMocks() (*command.MockCommandRunner, *auth.MockHTTPClient) {
	return new(command.MockCommandRunner), new(auth.MockHTTPClient)
}

func TestInstallDependency(t *testing.T) {
	mockRunner := new(command.MockCommandRunner)
	dep := "foo"
	flags := []string{"--index-url=" + IndexURL}
	mockRunner.On("Run", command.VirtualEnvPip, append([]string{"install", dep}, flags...)).Return(nil)

	err := installDependency(mockLogger, mockRunner, dep, flags...)

	assert.NoError(t, err)
	mockRunner.AssertExpectations(t)
}

func TestInstallDependencies(t *testing.T) {

	t.Run("successful without ArtifactRegistryUrl and RequirementsPath", func(t *testing.T) {
		mockRunner, mockClient := setupMocks()
		args := Arguments{
			Dependencies: []string{"foo", "bar"},
		}
		for _, dep := range args.Dependencies {
			indexFlags := []string{"--index-url=" + IndexURL}
			mockRunner.On("Run", command.VirtualEnvPip, append([]string{"install", dep}, indexFlags...)).Return(nil)
		}

		err := installDependencies(mockRunner, args, mockClient, mockLogger)

		assert.NoError(t, err)
		mockRunner.AssertExpectations(t)
	})

	t.Run("successful with Dependencies and ArtifactRegistryUrl", func(t *testing.T) {
		mockRunner, mockClient := setupMocks()
		args := Arguments{
			Dependencies:        []string{"foo", "bar"},
			ArtifactRegistryUrl: "https://foo-registry.com",
		}
		mockClient.On("Do", mock.Anything).Return(&http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"access_token": "foo-token"}`))),
		}, nil)
		mockAuthURL := "https://oauth2accesstoken:foo-token@foo-registry.com/simple"
		for _, dep := range args.Dependencies {
			indexFlags := []string{"--index-url=" + IndexURL, "--extra-index-url=" + mockAuthURL}
			mockRunner.On("Run", command.VirtualEnvPip, append([]string{"install", dep}, indexFlags...)).Return(nil)
		}

		err := installDependencies(mockRunner, args, mockClient, mockLogger)

		assert.Nil(t, err)
		mockRunner.AssertExpectations(t)
		mockClient.AssertExpectations(t)
	})

	t.Run("successful with Dependencies and RequirementsPath", func(t *testing.T) {
		mockRunner, mockClient := setupMocks()
		args := Arguments{
			Dependencies:     []string{"foo", "bar"},
			RequirementsPath: "requirements.txt",
		}
		indexFlags := []string{"--index-url=" + IndexURL}
		mockRunner.On("Run", command.VirtualEnvPip, append([]string{"install", "-r", args.RequirementsPath}, indexFlags...)).Return(nil)
		for _, dep := range args.Dependencies {
			mockRunner.On("Run", command.VirtualEnvPip, append([]string{"install", dep}, indexFlags...)).Return(nil)
		}

		err := installDependencies(mockRunner, args, mockClient, mockLogger)

		assert.NoError(t, err)
		mockRunner.AssertExpectations(t)
	})

	t.Run("successful with Dependencies, RequirementsPath, and ArtifactRegistryUrl", func(t *testing.T) {
		mockRunner, mockClient := setupMocks()
		args := Arguments{
			Dependencies:        []string{"foo", "bar"},
			RequirementsPath:    "requirements.txt",
			ArtifactRegistryUrl: "https://foo-registry.com",
		}
		mockClient.On("Do", mock.Anything).Return(&http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"access_token": "foo-token"}`))),
		}, nil)
		mockAuthURL := "https://oauth2accesstoken:foo-token@foo-registry.com/simple"
		indexFlags := []string{"--index-url=" + IndexURL, "--extra-index-url=" + mockAuthURL}
		mockRunner.On("Run", command.VirtualEnvPip, append([]string{"install", "-r", args.RequirementsPath}, indexFlags...)).Return(nil)
		for _, dep := range args.Dependencies {
			mockRunner.On("Run", command.VirtualEnvPip, append([]string{"install", dep}, indexFlags...)).Return(nil)
		}

		err := installDependencies(mockRunner, args, mockClient, mockLogger)

		assert.NoError(t, err)
		mockRunner.AssertExpectations(t)
	})

	t.Run("failed to get authenticated Artifact Registry URL", func(t *testing.T) {
		mockRunner, mockClient := setupMocks()
		args := Arguments{
			Dependencies:        []string{"foo", "bar"},
			ArtifactRegistryUrl: "https://foo-registry.com",
		}
		mockClient.On("Do", mock.Anything).Return(&http.Response{}, fmt.Errorf("mock http error"))

		err := installDependencies(mockRunner, args, mockClient, mockLogger)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mock http error")
		mockRunner.AssertExpectations(t)
	})

	t.Run("failed to install from requirements file", func(t *testing.T) {
		mockRunner, mockClient := setupMocks()
		args := Arguments{
			RequirementsPath: "requirements.txt",
		}
		indexFlags := []string{"--index-url=" + IndexURL}
		mockRunner.On("Run", command.VirtualEnvPip, append([]string{"install", "-r", args.RequirementsPath}, indexFlags...)).Return(
			fmt.Errorf("mock installation error"),
		)

		err := installDependencies(mockRunner, args, mockClient, mockLogger)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mock installation error")
		mockRunner.AssertExpectations(t)
	})

	t.Run("failed to install dependencies", func(t *testing.T) {
		mockRunner, mockClient := setupMocks()
		args := Arguments{
			Dependencies: []string{"foo"},
		}
		indexFlags := []string{"--index-url=" + IndexURL}
		for _, dep := range args.Dependencies {
			mockRunner.On("Run", command.VirtualEnvPip, append([]string{"install", dep}, indexFlags...)).Return(
				fmt.Errorf("mock installation error"),
			)
		}

		err := installDependencies(mockRunner, args, mockClient, mockLogger)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mock installation error")
		mockRunner.AssertExpectations(t)
	})
}

func TestParseArgs(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	t.Run("parse valid arguments", func(t *testing.T) {
		f := setupFlags()
		err := f.Parse([]string{"--command=test_command", "--dependencies=dep1 dep2", "--requirementsPath=req.txt", "--artifactRegistryUrl=https://test-url"})
		assert.NoError(t, err)

		args, err := ParseArgs(f)
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{"dep1", "dep2"}, args.Dependencies)
		assert.Equal(t, "req.txt", args.RequirementsPath)
		assert.Equal(t, "https://test-url", args.ArtifactRegistryUrl)
	})

	t.Run("successful with some arguments", func(t *testing.T) {
		f := setupFlags()
		err := f.Parse([]string{"--command=test_command", "--dependencies=dep1 dep2"})
		assert.NoError(t, err)

		args, err := ParseArgs(f)
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{"dep1", "dep2"}, args.Dependencies)
		assert.Equal(t, "", args.RequirementsPath)
		assert.Equal(t, "", args.ArtifactRegistryUrl)
	})
}

func setupFlags() *pflag.FlagSet {
	f := pflag.NewFlagSet("test", pflag.ContinueOnError)
	f.String("command", "", "")
	f.String("dependencies", "", "")
	f.String("requirementsPath", "", "")
	f.String("artifactRegistryUrl", "", "")
	f.Bool("verbose", false, "")
	f.String("script", "", "")
	return f
}

func TestExecute(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		mockRunner, mockClient := setupMocks()
		args := Arguments{
			Dependencies:     []string{"foo", "bar"},
			RequirementsPath: "requirements.txt",
		}
		mockRunner.On("Run", "python3", []string{"-m", "venv", "venv"}).Return(nil)
		indexFlags := []string{"--index-url=" + IndexURL}
		for _, dep := range args.Dependencies {
			mockRunner.On("Run", command.VirtualEnvPip, append([]string{"install", dep}, indexFlags...)).Return(nil)
		}
		mockRunner.On("Run", command.VirtualEnvPip, append([]string{"install", "-r", args.RequirementsPath}, indexFlags...)).Return(nil)

		err := Execute(mockRunner, args, mockClient)

		assert.NoError(t, err)
		mockRunner.AssertExpectations(t)
	})

	t.Run("failed to create virtual environment", func(t *testing.T) {
		mockRunner, mockClient := setupMocks()
		args := Arguments{
			Dependencies: []string{"foo", "bar"},
		}
		mockRunner.On("Run", "python3", []string{"-m", "venv", "venv"}).Return(fmt.Errorf("mock virtual environment creating error"))

		err := Execute(mockRunner, args, mockClient)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mock virtual environment creating error")
		mockRunner.AssertExpectations(t)
	})

	t.Run("failed to install dependencies", func(t *testing.T) {
		mockRunner, mockClient := setupMocks()
		args := Arguments{
			Dependencies: []string{"foo"},
		}
		mockRunner.On("Run", "python3", []string{"-m", "venv", "venv"}).Return(nil)
		indexFlags := []string{"--index-url=" + IndexURL}
		for _, dep := range args.Dependencies {
			mockRunner.On("Run", command.VirtualEnvPip, append([]string{"install", dep}, indexFlags...)).Return(fmt.Errorf("mock installation error"))
		}

		err := Execute(mockRunner, args, mockClient)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mock installation error")
		mockRunner.AssertExpectations(t)
	})
}
