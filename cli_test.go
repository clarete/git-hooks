package main

import (
	"fmt"
	"github.com/google/go-github/github"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Create temporary directory
func createDirectory(t *testing.T, dir string, context func(tempdir string)) {
	tempdir, err := ioutil.TempDir(dir, "git-hooks")
	assert.Nil(t, err)

	current, err := os.Getwd()
	assert.Nil(t, err)

	err = os.Chdir(tempdir)
	assert.Nil(t, err)

	context(tempdir)

	err = os.Chdir(current)
	assert.Nil(t, err)

	err = os.RemoveAll(tempdir)
	assert.Nil(t, err)
}

// Create temporary git repo
func createGitRepo(t *testing.T, context func(tempdir string)) {
	createDirectory(t, filepath.Join("fixtures", "repos"), func(tempdir string) {
		cmd := exec.Command("git", "init")
		err := cmd.Run()
		assert.Nil(t, err)

		context(tempdir)
	})
}

func TestList(t *testing.T) {
	gitExec(GIT["RemoveTemplateDir"])
	// not inside git repo
	// Should outside of this repo
	createDirectory(t, os.TempDir(), func(tempdir string) {
		list()
		assert.Equal(t, MESSAGES["NotGitRepo"], logger.infos[0])
		logger.clear()
	})

	// git hooks not installed
	createGitRepo(t, func(tempdir string) {
		list()
		assert.Equal(t, MESSAGES["NotInstalled"], logger.infos[0])
		logger.clear()
	})

	// git hooks installed
	createGitRepo(t, func(tempdir string) {
		cmd := exec.Command("git", "hooks", "install")
		err := cmd.Run()
		assert.Nil(t, err)

		list()
		assert.Equal(t, MESSAGES["Installed"], logger.infos[0])
		logger.clear()
	})
}

// Include uninstall test
func TestInstall(t *testing.T) {
	// not inside git repo
	createDirectory(t, os.TempDir(), func(tempdir string) {
		install(true)
		assert.Equal(t, MESSAGES["NotGitRepo"], logger.errors[0])
		logger.clear()
	})

	// installed
	createGitRepo(t, func(tempdir string) {
		install(true)
		assert.Equal(t, len(TRIGGERS)*2, len(logger.infos)) // with newline
		logger.clear()
	})

	// already installed
	createGitRepo(t, func(tempdir string) {
		install(true)
		install(true)
		assert.Equal(t, MESSAGES["ExistHooks"], logger.errors[0])
		logger.clear()
	})

	// uninstall
	createGitRepo(t, func(tempdir string) {
		install(true)
		logger.clear()

		uninstall()
		assert.Equal(t, MESSAGES["Restore"], logger.infos[0])
		logger.clear()
	})

	// not installed
	createGitRepo(t, func(tempdir string) {
		uninstall()
		assert.Equal(t, MESSAGES["NotExistHooks"], logger.errors[0])
		logger.clear()
	})
}

func TestInstallGlobal(t *testing.T) {
	// backup current configuration file
	templatedir, err := gitExec(GIT["GetTemplateDir"])

	createDirectory(t, os.TempDir(), func(tempdir string) {
		DIRS["HomeTemplate"] = filepath.Join(tempdir, "home")

		installGlobal(tempdir)
		newTemplatedir, err := gitExec(GIT["GetTemplateDir"])
		assert.Nil(t, err)
		assert.Equal(t, DIRS["HomeTemplate"], newTemplatedir)
		assert.Equal(t, 0, strings.Index(logger.infos[len(logger.infos)-2].(string), MESSAGES["SetTemplateDir"]))
		logger.clear()
	})

	// already installed
	createDirectory(t, os.TempDir(), func(tempdir string) {
		DIRS["HomeTemplate"] = filepath.Join(tempdir, "home")

		installGlobal(tempdir)
		logger.clear()

		installGlobal(tempdir)
		newTemplatedir, err := gitExec(GIT["GetTemplateDir"])
		assert.Nil(t, err)
		assert.Equal(t, DIRS["HomeTemplate"], newTemplatedir)
		assert.Equal(t, 0, strings.Index(logger.infos[0].(string), MESSAGES["SetTemplateDir"]))
		logger.clear()
	})

	// restore
	if err == nil {
		gitExec(GIT["SetTemplateDir"] + templatedir)
	} else {
		gitExec(GIT["RemoveTemplateDir"])
	}
	fmt.Println(logger.infos)
}

func TestUninstallGlobal(t *testing.T) {
	// backup current configuration file
	templatedir, err := gitExec(GIT["GetTemplateDir"])

	createDirectory(t, os.TempDir(), func(tempdir string) {
		DIRS["HomeTemplate"] = filepath.Join(tempdir, "home")

		installGlobal(tempdir)
		newTemplatedir, err := gitExec(GIT["GetTemplateDir"])
		assert.Nil(t, err)
		assert.Equal(t, DIRS["HomeTemplate"], newTemplatedir)
		assert.Equal(t, 0, strings.Index(logger.infos[len(logger.infos)-2].(string), MESSAGES["SetTemplateDir"]))
		logger.clear()

		uninstallGlobal()
		newTemplatedir, err = gitExec(GIT["GetTemplateDir"])
		assert.NotNil(t, err)
	})

	// not installed
	createDirectory(t, os.TempDir(), func(tempdir string) {
		uninstallGlobal()
		newTemplatedir, err := gitExec(GIT["GetTemplateDir"])
		assert.NotNil(t, err)
		assert.Equal(t, "", newTemplatedir)
	})

	// restore
	if err == nil {
		gitExec(GIT["SetTemplateDir"] + templatedir)
	} else {
		gitExec(GIT["RemoveTemplateDir"])
	}
}

func TestUpdate(t *testing.T) {
	content := "Hello, client"
	// start test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, content)
	}))
	defer ts.Close()

	tagName := "v1.0.0"
	assetName := "git-hooks_darwin_386.tar.gz"
	releases := []github.RepositoryRelease{
		github.RepositoryRelease{
			TagName: &tagName,
			Assets: []github.ReleaseAsset{
				github.ReleaseAsset{
					Name:               &assetName,
					BrowserDownloadURL: &ts.URL,
				},
			},
		},
	}

	fmt.Println(releases)
}

func TestIdentity(t *testing.T) {
	createGitRepo(t, func(tempdir string) {
		identity()
		assert.True(t, len(logger.errors) != 0)
		logger.clear()
	})

	createGitRepo(t, func(tempdir string) {
		cmd := exec.Command("bash", "-c", "touch a;git add a;git commit -m 'test';")
		err := cmd.Run()
		assert.Nil(t, err)

		identity()
		assert.True(t, len(logger.errors) == 0)
		logger.clear()
	})
}

func TestRun(t *testing.T) {
}
