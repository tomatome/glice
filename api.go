package glice

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gocolly/colly"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type licenseFormat struct {
	name  string
	color color.Attribute
}

var licenseCol = map[string]licenseFormat{
	"other":      {name: "Other", color: color.FgBlue},
	"mit":        {name: "MIT", color: color.FgGreen},
	"lgpl-3.0":   {name: "LGPL-3.0", color: color.FgCyan},
	"mpl-2.0":    {name: "MPL-2.0", color: color.FgHiBlue},
	"agpl-3.0":   {name: "AGPL-3.0", color: color.FgHiCyan},
	"unlicense":  {name: "Unlicense", color: color.FgHiRed},
	"apache-2.0": {name: "Apache-2.0", color: color.FgHiGreen},
	"gpl-3.0":    {name: "GPL-3.0", color: color.FgHiMagenta},
}
var licenseColMap = map[string]color.Attribute{
	"mit":          color.FgGreen,
	"apache-2.0":   color.FgHiGreen,
	"gpl-2.0":      color.FgMagenta,
	"gpl-3.0":      color.FgHiMagenta,
	"lgpl-2.1":     color.FgCyan,
	"lgpl-3.0":     color.FgHiCyan,
	"mpl-2.0":      color.FgHiBlue,
	"bsd-2-clause": color.FgYellow,
	"bsd-3-clause": color.FgHiYellow,
	"epl-2.0":      color.FgRed,
	"artistic-2.0": color.FgHiRed,
	"bsl-1.0":      color.FgBlue,
	"cc0-1.0":      color.FgHiWhite,
	"unlicense":    color.FgHiRed,
	"agpl-3.0":     color.FgHiCyan,
	"other":        color.FgBlue,
}

func getLicenseColor(license string) color.Attribute {
	license = strings.ToLower(license)
	if color, ok := licenseColMap[license]; ok {
		return color
	}
	return color.FgYellow // 默认颜色
}

// Repository holds information about the repository
type Repository struct {
	Name      string `json:"name,omitempty"`
	Shortname string `json:"-"`
	URL       string `json:"url,omitempty"`
	Host      string `json:"host,omitempty"`
	Author    string `json:"author,omitempty"`
	Project   string `json:"project,omitempty"`
	Text      string `json:"-"`
	License   string `json:"license"`
	Version   string `json:"Version"`
}

func newGitClient(c context.Context, keys map[string]string, star bool) *gitClient {
	var tc *http.Client
	var ghLogged bool
	if v := keys["github.com"]; v != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: v},
		)
		tc = oauth2.NewClient(c, ts)
		ghLogged = true
	}
	return &gitClient{
		gh: githubClient{
			Client: github.NewClient(tc),
			logged: ghLogged,
		},
		star: star,
	}
}

type gitClient struct {
	gh   githubClient
	star bool
}

type githubClient struct {
	*github.Client
	logged bool
}

// GetLicense for a repository
func (gc *gitClient) GetLicense(ctx context.Context, r *Repository) error {
	switch r.Host {
	case "github.com":
		rl, _, err := gc.gh.Repositories.License(ctx, r.Author, r.Project)
		if err != nil {
			return err
		}

		name, clr := licenseCol[*rl.License.Key].name, licenseCol[*rl.License.Key].color
		if name == "" {
			name = *rl.License.Key
			clr = color.FgYellow
		}
		r.Shortname = color.New(clr).Sprintf(name)
		r.License = name
		r.Text = rl.GetContent()

		if gc.star && gc.gh.logged {
			gc.gh.Activity.Star(ctx, r.Author, r.Project)
		}
	case "pkg.go.dev":
		c := colly.NewCollector(
			colly.MaxDepth(2),
			colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3"),
		)
		c.SetRequestTimeout(10 * time.Second)

		c.OnHTML("span[data-test-id=\"UnitHeader-version\"]", func(e *colly.HTMLElement) {
			version := e.ChildText("a")
			version = version[9:]
			version = strings.Split(version, "G")[0]
			version = strings.TrimSpace(version)
			if !strings.EqualFold(r.Version, version) {
				r.Version = fmt.Sprintf("%s (!new:%s)", r.Version, version)
			}
		})
		c.OnHTML("span[data-test-id=\"UnitHeader-licenses\"]", func(e *colly.HTMLElement) {
			license := e.ChildText("a")
			r.Shortname = color.New(getLicenseColor(license)).Sprintf(license)
		})
		c.OnHTML(".UnitMeta-repo", func(e *colly.HTMLElement) {
			repo := e.ChildText("a")
			r.Project = repo
		})

		err := c.Visit(r.URL)
		if err != nil {
			fmt.Println(r.URL, "error:", err)
		}
	}

	return nil
}
