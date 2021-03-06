package source

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/skygeario/skygear-server/migrate/pkg/migrate/logging"
)

type Github struct {
	CacheDir string
}

type config struct {
	Owner string
	Repo  string
	Path  string
	Ref   string
}

func (g *Github) Download(sourceURL string) (string, error) {
	c, err := g.parse(sourceURL)
	if err != nil {
		return "", fmt.Errorf("invalid github source url: %s", err.Error())
	}

	return g.downloadFromClone(c)
}

func (g *Github) parse(sourceURL string) (*config, error) {
	c := &config{}
	u, err := url.Parse(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("invalid source url: %s", err.Error())
	}

	c.Owner = u.Host
	p := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(p) < 1 {
		return nil, fmt.Errorf("invalid source url: missing host")
	}
	c.Repo = p[0]
	c.Path = strings.Join(p[1:], "/")
	c.Ref = u.Fragment

	return c, nil
}

func (g *Github) downloadFromClone(c *config) (string, error) {
	sourceCodeDirPath := g.getSourceCodeDirPath(g.CacheDir, c, "clone")
	cloneSource := fmt.Sprintf(
		"https://github.com/%s/%s.git",
		c.Owner,
		c.Repo,
	)

	logger := logrus.New()
	logwriter := logging.NewLogWriter(logger, logrus.InfoLevel)
	if _, e := os.Stat(sourceCodeDirPath); os.IsNotExist(e) {
		cmd := exec.Command("git", "clone", cloneSource, sourceCodeDirPath)
		logger.WithFields(logrus.Fields{
			"command": cmd.String(),
		}).Info("execute command")
		// git clone pipe its progress message to stderr, but it is not error
		// so we use same writer with info level to both stdout and stderror
		cmd.Stdout = logwriter
		cmd.Stderr = logwriter
		err := cmd.Run()
		if err != nil {
			return "", fmt.Errorf("failed to git clone source: %v", err.Error())
		}

		// check out references
		cmd = exec.Command("git", "-C", sourceCodeDirPath, "checkout", c.Ref)
		logger.WithFields(logrus.Fields{
			"command": cmd.String(),
		}).Info("execute command")
		cmd.Stdout = logwriter
		cmd.Stderr = logwriter
		err = cmd.Run()
		if err != nil {
			return "", fmt.Errorf("failed to git checkout source: %v", err.Error())
		}
	}

	migrateSrcDirPath := filepath.Join(sourceCodeDirPath, c.Path)
	return fmt.Sprintf("file://%s", migrateSrcDirPath), nil
}

func (g *Github) getSourceCodeDirPath(tmpDir string, c *config, subfix string) string {
	return filepath.Join(
		tmpDir,
		fmt.Sprintf("%s-%s-%s-%s", c.Owner, c.Repo, c.Ref, subfix),
	)
}
