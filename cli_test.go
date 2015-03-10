package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Create temporary directory
func createDirectory(t *testing.T, dir string, context func(tempdir string)) {
	name, err := ioutil.TempDir(dir, "git-hooks")
	assert.Nil(t, err)

	wd, err := os.Getwd()
	assert.Nil(t, err)

	err = os.Chdir(name)
	assert.Nil(t, err)

	context(name)

	err = os.Chdir(wd)
	assert.Nil(t, err)

	err = os.RemoveAll(name)
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

	createGitRepo(t, func(tempdir string) {
		install(true)
		assert.Equal(t, len(TRIGGERS)*2, len(logger.infos)) // with newline

		logger.clear()
	})

	createGitRepo(t, func(tempdir string) {
		wd, _ := os.Getwd()
		fmt.Println(wd)

		install(true)
		install(true)
		assert.Equal(t, MESSAGES["ExistHooks"], logger.errors[0])

		logger.clear()
	})

	createGitRepo(t, func(tempdir string) {
		install(true)
		logger.clear()

		uninstall()

		assert.Equal(t, MESSAGES["Restore"], logger.infos[0])

		logger.clear()
	})

	createGitRepo(t, func(tempdir string) {
		uninstall()
		assert.Equal(t, MESSAGES["NotExistHooks"], logger.errors[0])

		logger.clear()
	})
}

func TestInstallGlobal(t *testing.T) {
	createDirectory(t, os.TempDir(), func(tempdir string) {
		//globalTemplate := DIRS["GlobalTemplate"]
		DIRS["GlobalTemplate"] = filepath.Join(tempdir, "global")
		//homeTemplate := DIRS["HomeTemplate"]
		DIRS["HomeTemplate"] = filepath.Join(tempdir, "home")

		installGlobal(tempdir)
		assert.Equal(t, 0, strings.Index(cmds[0], GIT["SetTemplateDir"]))
		assert.Equal(t, 0, strings.Index(logger.infos[0].(string), MESSAGES["SetTemplateDir"]))

		logger.clear()
	})
}
