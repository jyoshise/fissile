package compilator

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcloud/fissile/docker"
	"github.com/hpcloud/fissile/model"
	"github.com/hpcloud/fissile/scripts/compilation"
	"github.com/hpcloud/fissile/util"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	dockerImageEnvVar      = "FISSILE_TEST_DOCKER_IMAGE"
	defaultDockerTestImage = "ubuntu:14.04"
)

var dockerImageName string

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)

	dockerImageName = os.Getenv(dockerImageEnvVar)
	if dockerImageName == "" {
		dockerImageName = defaultDockerTestImage
	}

	retCode := m.Run()

	os.Exit(retCode)
}

func TestCompilation(t *testing.T) {
}

func TestCompilationSourcePreparation(t *testing.T) {
}

func TestGetPackageStatusCompiled(t *testing.T) {
	assert := assert.New(t)

	compilationWorkDir, err := util.TempDir("", "fissile-tests")
	assert.Nil(err)

	dockerManager, err := docker.NewImageManager()
	assert.Nil(err)

	workDir, err := os.Getwd()
	ntpReleasePath := filepath.Join(workDir, "../test-assets/ntp-release-2")
	release, err := model.NewRelease(ntpReleasePath)
	assert.Nil(err)

	compilator, err := NewCompilator(dockerManager, compilationWorkDir, "fissile-test", compilation.FakeBase, "3.14.15")
	assert.Nil(err)

	compilator.initPackageMaps(release)

	compiledPackagePath := filepath.Join(compilationWorkDir, release.Packages[0].Name, release.Packages[0].Fingerprint, "compiled")
	err = os.MkdirAll(compiledPackagePath, 0755)
	assert.Nil(err)

	err = ioutil.WriteFile(filepath.Join(compiledPackagePath, "foo"), []byte{}, 0700)
	assert.Nil(err)

	status, err := compilator.getPackageStatus(release.Packages[0])

	assert.Nil(err)
	assert.Equal(packageCompiled, status)
}

func TestGetPackageStatusNone(t *testing.T) {
	assert := assert.New(t)

	compilationWorkDir, err := util.TempDir("", "fissile-tests")
	assert.Nil(err)

	dockerManager, err := docker.NewImageManager()
	assert.Nil(err)

	workDir, err := os.Getwd()
	ntpReleasePath := filepath.Join(workDir, "../test-assets/ntp-release-2")
	release, err := model.NewRelease(ntpReleasePath)
	assert.Nil(err)

	compilator, err := NewCompilator(dockerManager, compilationWorkDir, "fissile-test", compilation.FakeBase, "3.14.15")
	assert.Nil(err)

	compilator.initPackageMaps(release)

	status, err := compilator.getPackageStatus(release.Packages[0])

	assert.Nil(err)
	assert.Equal(packageNone, status)
}

func TestPackageFolderStructure(t *testing.T) {
	assert := assert.New(t)

	compilationWorkDir, err := util.TempDir("", "fissile-tests")
	assert.Nil(err)

	dockerManager, err := docker.NewImageManager()
	assert.Nil(err)

	workDir, err := os.Getwd()
	ntpReleasePath := filepath.Join(workDir, "../test-assets/ntp-release-2")
	release, err := model.NewRelease(ntpReleasePath)
	assert.Nil(err)

	compilator, err := NewCompilator(dockerManager, compilationWorkDir, "fissile-test", compilation.FakeBase, "3.14.15")
	assert.Nil(err)

	err = compilator.createCompilationDirStructure(release.Packages[0])
	assert.Nil(err)

	exists, err := validatePath(compilator.getDependenciesPackageDir(release.Packages[0]), true, "")
	assert.Nil(err)
	assert.True(exists)

	exists, err = validatePath(compilator.getSourcePackageDir(release.Packages[0]), true, "")
	assert.Nil(err)
	assert.True(exists)
}

func TestPackageDependenciesPreparation(t *testing.T) {
	assert := assert.New(t)

	compilationWorkDir, err := util.TempDir("", "fissile-tests")
	assert.Nil(err)

	dockerManager, err := docker.NewImageManager()
	assert.Nil(err)

	workDir, err := os.Getwd()
	ntpReleasePath := filepath.Join(workDir, "../test-assets/tor-boshrelease-0.3.5")
	release, err := model.NewRelease(ntpReleasePath)
	assert.Nil(err)

	compilator, err := NewCompilator(dockerManager, compilationWorkDir, "fissile-test", compilation.FakeBase, "3.14.15")
	assert.Nil(err)

	pkg, err := release.LookupPackage("tor")
	assert.Nil(err)
	err = compilator.createCompilationDirStructure(pkg)
	assert.Nil(err)
	err = os.MkdirAll(compilator.getPackageCompiledDir(pkg.Dependencies[0]), 0755)
	assert.Nil(err)

	dummyCompiledFile := filepath.Join(compilator.getPackageCompiledDir(pkg.Dependencies[0]), "foo")
	file, err := os.Create(dummyCompiledFile)
	assert.Nil(err)
	file.Close()

	err = compilator.copyDependencies(pkg)
	assert.Nil(err)

	expectedDummyFileLocation := filepath.Join(compilator.getDependenciesPackageDir(pkg), pkg.Dependencies[0].Name, "foo")
	exists, err := validatePath(expectedDummyFileLocation, false, "")
	assert.Nil(err)
	assert.True(exists, expectedDummyFileLocation)
}

func TestCompilePackage(t *testing.T) {
	assert := assert.New(t)

	compilationWorkDir, err := util.TempDir("", "fissile-tests")
	assert.Nil(err)

	dockerManager, err := docker.NewImageManager()
	assert.Nil(err)

	workDir, err := os.Getwd()
	ntpReleasePath := filepath.Join(workDir, "../test-assets/ntp-release-2")
	release, err := model.NewRelease(ntpReleasePath)
	assert.Nil(err)

	testRepository := fmt.Sprintf("fissile-test-%s", uuid.New())

	comp, err := NewCompilator(dockerManager, compilationWorkDir, testRepository, compilation.FakeBase, "3.14.15")
	assert.Nil(err)

	imageName := comp.BaseImageName()

	_, err = comp.CreateCompilationBase(dockerImageName)
	defer func() {
		err = dockerManager.RemoveImage(imageName)
		assert.Nil(err)
	}()
	assert.Nil(err)

	compiled, err := comp.compilePackage(release.Packages[0])
	assert.Nil(err)
	assert.True(compiled)
}
