package gotagger

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	sgit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sassoftware.io/clis/gotagger/internal/git"
	"sassoftware.io/clis/gotagger/internal/testutils"
)

type setupRepoFunc func(testutils.T, *sgit.Repository, string)

func TestGotagger_getLatest(t *testing.T) {
	tests := []struct {
		title    string
		prefix   string
		module   module
		repoFunc setupRepoFunc
		want     string
	}{
		{
			title:    "no latest",
			prefix:   "v",
			module:   module{".", "foo", ""},
			repoFunc: simpleGoRepo,
			want:     "v1.0.0",
		},
		{
			title:    "sub module",
			prefix:   "v",
			module:   module{filepath.Join("sub", "module"), "foo/sub/module", "sub/module/"},
			repoFunc: simpleGoRepo,
			want:     "v0.1.0",
		},
		{
			title:    "latest foo v1 directory",
			prefix:   "v",
			module:   module{".", "foo", ""},
			repoFunc: v2DirGitRepo,
			want:     "v1.0.0",
		},
		{
			title:    "latest bar v1 directory",
			prefix:   "v",
			module:   module{"bar", "foo/bar", "bar/"},
			repoFunc: v2DirGitRepo,
			want:     "v1.0.0",
		},
		{
			title:    "latest foo v2 directory",
			prefix:   "v",
			module:   module{"v2", "foo/v2", ""},
			repoFunc: v2DirGitRepo,
			want:     "v2.0.0",
		},
		{
			title:    "latest foo/bar v2 directory",
			prefix:   "v",
			module:   module{filepath.Join("bar", "v2"), "foo/bar/v2", "bar/"},
			repoFunc: v2DirGitRepo,
			want:     "v2.0.0",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.title, func(t *testing.T) {
			t.Parallel()

			g, repo, path, teardown := newGotagger(t)
			defer teardown()

			tt.repoFunc(t, repo, path)

			g.Config.VersionPrefix = tt.prefix
			if got, _, err := g.getLatest(tt.module); assert.NoError(t, err) {
				assert.Equal(t, tt.want, got.Original())
			}
		})
	}
}

func TestGotagger_ModuleVersion(t *testing.T) {
	g, repo, path, teardown := newGotagger(t)
	defer teardown()

	simpleGoRepo(t, repo, path)

	if v, err := g.ModuleVersions("foo/sub/module"); assert.NoError(t, err) {
		assert.Equal(t, []string{"sub/module/v0.1.1"}, v)
	}
}

func TestGotagger_TagRepo(t *testing.T) {
	tests := []struct {
		title    string
		prefix   string
		repoFunc setupRepoFunc
		message  string
		files    []testutils.FileCommit
		want     []string
	}{
		{
			title:    "v-prefix tags",
			prefix:   "v",
			repoFunc: mixedTagRepo,
			message:  "release: the foos\n",
			files: []testutils.FileCommit{
				{
					Path:     "CHANGELOG.md",
					Contents: []byte("# Foo Change Log\n"),
				},
			},
			want: []string{"v1.1.0"},
		},
		{
			title:    "unprefixed tags",
			prefix:   "",
			repoFunc: mixedTagRepo,
			message:  "release: the bars\n",
			files: []testutils.FileCommit{
				{
					Path:     "CHANGELOG.md",
					Contents: []byte("# Bar Change Log\n"),
				},
			},
			want: []string{"0.1.1"},
		},
		{
			title:  "release root v1 on master implicit",
			prefix: "v",
			repoFunc: func(t testutils.T, r *sgit.Repository, p string) {
				masterV1GitRepo(t, r, p)

				testutils.CommitFile(t, r, p, "foo.go", "feat: add foo.go", []byte("foo\n"))
			},
			message: "release: the foos\n",
			files: []testutils.FileCommit{
				{
					Path:     "CHANGELOG.md",
					Contents: []byte("# Foo Change Log\n"),
				},
			},
			want: []string{"v1.1.0"},
		},
		{
			title:  "release root v1 on master explicit",
			prefix: "v",
			repoFunc: func(t testutils.T, r *sgit.Repository, p string) {
				masterV1GitRepo(t, r, p)

				testutils.CommitFile(t, r, p, "foo.go", "feat: add foo.go", []byte("foo\n"))
			},
			message: "release: the foos\n\nModules: foo\n",
			files: []testutils.FileCommit{
				{
					Path:     "CHANGELOG.md",
					Contents: []byte("# Foo Change Log\n"),
				},
			},
			want: []string{"v1.1.0"},
		},
		{
			title:  "release bar v1 on master",
			prefix: "v",
			repoFunc: func(t testutils.T, r *sgit.Repository, p string) {
				masterV1GitRepo(t, r, p)

				testutils.CommitFile(t, r, p, filepath.Join("bar", "bar.go"), "feat: add bar/bar.go", []byte("bar\n"))
			},
			message: "release: the bars\n\nModules: foo/bar",
			files: []testutils.FileCommit{
				{
					Path:     filepath.Join("bar", "CHANGELOG.md"),
					Contents: []byte("# Bar Change Log\n"),
				},
			},
			want: []string{"bar/v1.1.0"},
		},
		{
			title:  "release all v1 on master",
			prefix: "v",
			repoFunc: func(t testutils.T, r *sgit.Repository, p string) {
				masterV1GitRepo(t, r, p)

				testutils.CommitFile(t, r, p, "foo.go", "feat: add foo.go", []byte("foo\n"))
				testutils.CommitFile(t, r, p, filepath.Join("bar", "bar.go"), "feat: add bar/bar.go", []byte("bar\n"))
			},
			message: "release: all the things\n\nModules: foo, foo/bar",
			files: []testutils.FileCommit{
				{
					Path:     "CHANGELOG.md",
					Contents: []byte("# Foo Change Log\n"),
				},
				{
					Path:     filepath.Join("bar", "CHANGELOG.md"),
					Contents: []byte("# Bar Change Log\n"),
				},
			},
			want: []string{"v1.1.0", "bar/v1.1.0"},
		},
		{
			title:  "release root v2 on master implicit",
			prefix: "v",
			repoFunc: func(t testutils.T, r *sgit.Repository, p string) {
				masterV2GitRepo(t, r, p)

				testutils.CommitFile(t, r, p, "foo.go", "feat: add foo.go", []byte("foo\n"))
			},
			message: "release: the foos\n",
			files: []testutils.FileCommit{
				{
					Path:     "CHANGELOG.md",
					Contents: []byte("# Foo Change Log\n"),
				},
			},
			want: []string{"v2.1.0"},
		},
		{
			title:  "release root v2 on master explicit",
			prefix: "v",
			repoFunc: func(t testutils.T, r *sgit.Repository, p string) {
				masterV2GitRepo(t, r, p)

				testutils.CommitFile(t, r, p, "foo.go", "feat: add foo.go", []byte("foo\n"))
			},
			message: "release: the foos\n\nModules: foo/v2\n",
			files: []testutils.FileCommit{
				{
					Path:     "CHANGELOG.md",
					Contents: []byte("# Foo Change Log\n"),
				},
			},
			want: []string{"v2.1.0"},
		},
		{
			title:  "release bar v2 on master",
			prefix: "v",
			repoFunc: func(t testutils.T, r *sgit.Repository, p string) {
				masterV2GitRepo(t, r, p)

				testutils.CommitFile(t, r, p, filepath.Join("bar", "bar.go"), "feat: add bar/bar.go", []byte("bar\n"))
			},
			message: "release: the bars\n\nModules: foo/bar/v2",
			files: []testutils.FileCommit{
				{
					Path:     filepath.Join("bar", "CHANGELOG.md"),
					Contents: []byte("# Bar Change Log\n"),
				},
			},
			want: []string{"bar/v2.1.0"},
		},
		{
			title:  "release all v2 on master",
			prefix: "v",
			repoFunc: func(t testutils.T, r *sgit.Repository, p string) {
				masterV2GitRepo(t, r, p)

				testutils.CommitFile(t, r, p, "foo.go", "feat: add foo.go", []byte("foo\n"))
				testutils.CommitFile(t, r, p, filepath.Join("bar", "bar.go"), "feat: add bar/bar.go", []byte("bar\n"))
			},
			message: "release: all the things\n\nModules: foo/bar/v2, foo/v2",
			files: []testutils.FileCommit{
				{
					Path:     "CHANGELOG.md",
					Contents: []byte("# Foo Change Log\n"),
				},
				{
					Path:     filepath.Join("bar", "CHANGELOG.md"),
					Contents: []byte("# Bar Change Log\n"),
				},
			},
			want: []string{"bar/v2.1.0", "v2.1.0"},
		},
		{
			title:  "release foo v1 implicit directory",
			prefix: "v",
			repoFunc: func(t testutils.T, repo *sgit.Repository, path string) {
				v2DirGitRepo(t, repo, path)

				// update foo
				testutils.CommitFile(t, repo, path, "foo.go", "feat: add foo.go\n", []byte("foo\n"))
			},
			message: "release: the foos\n",
			files: []testutils.FileCommit{
				{
					Path:     "CHANGELOG.md",
					Contents: []byte("# Foo Change Log\n"),
				},
			},
			want: []string{"v1.1.0"},
		},
		{
			title:  "release foo v1 explicit directory",
			prefix: "v",
			repoFunc: func(t testutils.T, repo *sgit.Repository, path string) {
				v2DirGitRepo(t, repo, path)

				// update foo/v2
				testutils.CommitFile(t, repo, path, "foo.go", "feat: add foo.go\n", []byte("foo\n"))
			},
			message: "release: the foos\n\nModules: foo\n",
			files: []testutils.FileCommit{
				{
					Path:     "CHANGELOG.md",
					Contents: []byte("# Foo Change Log\n"),
				},
			},
			want: []string{"v2.1.0"},
		},
		{
			title:  "release foo v2 implicit directory",
			prefix: "v",
			repoFunc: func(t testutils.T, r *sgit.Repository, p string) {
				v2DirGitRepo(t, r, p)

				testutils.CommitFile(t, r, p, filepath.Join("v2", "foo.go"), "feat: add v2/foo.go", []byte("foo\n"))
			},
			message: "release: the foos\n",
			files: []testutils.FileCommit{
				{
					Path:     filepath.Join("v2", "CHANGELOG.md"),
					Contents: []byte("# Foo Change Log\n"),
				},
			},
			want: []string{"v2.1.0"},
		},
		{
			title:  "release foo v2 explicit directory",
			prefix: "v",
			repoFunc: func(t testutils.T, r *sgit.Repository, p string) {
				v2DirGitRepo(t, r, p)

				testutils.CommitFile(t, r, p, filepath.Join("v2", "foo.go"), "feat: add v2/foo.go", []byte("foo\n"))
			},
			message: "release: the foos\n\nModules: foo/v2\n",
			files: []testutils.FileCommit{
				{
					Path:     filepath.Join("v2", "CHANGELOG.md"),
					Contents: []byte("# Foo Change Log\n"),
				},
			},
			want: []string{"v2.1.0"},
		},
		{
			title:  "release bar v1 directory",
			prefix: "v",
			repoFunc: func(t testutils.T, r *sgit.Repository, p string) {
				v2DirGitRepo(t, r, p)

				testutils.CommitFile(t, r, p, filepath.Join("bar", "bar.go"), "feat: add bar/bar.go", []byte("bar\n"))
			},
			message: "release: the bars\n\nModules: foo/bar\n",
			files: []testutils.FileCommit{
				{
					Path:     filepath.Join("bar", "CHANGELOG.md"),
					Contents: []byte("# Bar Change Log\n"),
				},
			},
			want: []string{"bar/v1.1.0"},
		},
		{
			title:  "release bar v2 directory",
			prefix: "v",
			repoFunc: func(t testutils.T, r *sgit.Repository, p string) {
				v2DirGitRepo(t, r, p)

				testutils.CommitFile(t, r, p, filepath.Join("bar", "v2", "bar.go"), "feat: add bar/v2/bar.go", []byte("bar\n"))
			},
			message: "release: the bars\n\nModules: foo/bar/v2\n",
			files: []testutils.FileCommit{
				{
					Path:     filepath.Join("bar", "v2", "CHANGELOG.md"),
					Contents: []byte("# Bar Change Log\n"),
				},
			},
			want: []string{"bar/v2.1.0"},
		},
		{
			title:  "release all v1 directory",
			prefix: "v",
			repoFunc: func(t testutils.T, r *sgit.Repository, p string) {
				v2DirGitRepo(t, r, p)

				testutils.CommitFile(t, r, p, "foo.go", "feat: add foo.go", []byte("foo\n"))
				testutils.CommitFile(t, r, p, filepath.Join("bar", "bar.go"), "feat: add bar/bar.go", []byte("bar\n"))
			},
			message: "release: all the v1 things\n\nModules: foo, foo/bar",
			files: []testutils.FileCommit{
				{
					Path:     "CHANGELOG.md",
					Contents: []byte("# Foo Change Log\n"),
				},
				{
					Path:     filepath.Join("bar", "CHANGELOG.md"),
					Contents: []byte("# Bar Change Log\n"),
				},
			},
			want: []string{"v1.1.0", "bar/v1.1.0"},
		},
		{
			title:  "release all v2 directory",
			prefix: "v",
			repoFunc: func(t testutils.T, r *sgit.Repository, p string) {
				v2DirGitRepo(t, r, p)

				testutils.CommitFile(t, r, p, filepath.Join("v2", "foo.go"), "feat: add v2/foo.go", []byte("foo\n"))
				testutils.CommitFile(t, r, p, filepath.Join("bar", "v2", "bar.go"), "feat: add bar/v2/bar.go", []byte("bar\n"))
			},
			message: "release: all the v2 things\n\nModules: foo/v2, foo/bar/v2",
			files: []testutils.FileCommit{
				{
					Path:     filepath.Join("v2", "CHANGELOG.md"),
					Contents: []byte("# Foo Change Log\n"),
				},
				{
					Path:     filepath.Join("bar", "v2", "CHANGELOG.md"),
					Contents: []byte("# Bar Change Log\n"),
				},
			},
			want: []string{"v2.1.0", "bar/v2.1.0"},
		},
		{
			title:  "release all directory",
			prefix: "v",
			repoFunc: func(t testutils.T, r *sgit.Repository, p string) {
				v2DirGitRepo(t, r, p)

				testutils.CommitFile(t, r, p, "foo.go", "feat: add foo.go", []byte("foo\n"))
				testutils.CommitFile(t, r, p, filepath.Join("bar", "bar.go"), "feat: add bar/bar.go", []byte("bar\n"))

				testutils.CommitFile(t, r, p, filepath.Join("v2", "foo.go"), "feat: add v2/foo.go", []byte("foo\n"))
				testutils.CommitFile(t, r, p, filepath.Join("bar", "v2", "bar.go"), "feat: add bar/v2/bar.go", []byte("bar\n"))
			},
			message: "release: all the things\n\nModules: foo, foo/bar, foo/v2, foo/bar/v2\n",
			files: []testutils.FileCommit{
				{
					Path:     "CHANGELOG.md",
					Contents: []byte("# Foo Change Log\n"),
				},
				{
					Path:     filepath.Join("bar", "CHANGELOG.md"),
					Contents: []byte("# Bar Change Log\n"),
				},
				{
					Path:     filepath.Join("v2", "CHANGELOG.md"),
					Contents: []byte("# Foo Change Log\n"),
				},
				{
					Path:     filepath.Join("bar", "v2", "CHANGELOG.md"),
					Contents: []byte("# Bar Change Log\n"),
				},
			},
			want: []string{"v1.1.0", "bar/v1.1.0", "v2.1.0", "bar/v2.1.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			t.Parallel()

			g, repo, path, teardown := newGotagger(t)
			defer teardown()

			tt.repoFunc(t, repo, path)

			// create a release commit
			testutils.CommitFiles(t, repo, path, tt.message, tt.files)

			g.Config.VersionPrefix = tt.prefix
			if versions, err := g.TagRepo(); assert.NoError(t, err) {
				assert.Equal(t, tt.want, versions)
			}
		})
	}
}

func TestGotagger_TagRepo_validation_extra(t *testing.T) {
	g, repo, path, teardown := newGotagger(t)
	defer teardown()

	masterV1GitRepo(t, repo, path)

	commitMsg := `release: extra module

Modules: foo/bar, foo
`
	testutils.CommitFile(t, repo, path, "CHANGELOG.md", commitMsg, []byte(`changes`))

	g.Config.CreateTag = true
	_, err := g.TagRepo()
	assert.EqualError(t, err, "module validation failed:\nmodules not changed by commit: foo/bar")
}

func TestGotagger_TagRepo_validation_missing(t *testing.T) {
	g, repo, path, teardown := newGotagger(t)
	defer teardown()

	masterV1GitRepo(t, repo, path)

	if err := ioutil.WriteFile(filepath.Join(path, "CHANGELOG.md"), []byte(`contents`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(filepath.Join(path, "bar", "CHANGELOG.md"), []byte(`contents`), 0600); err != nil {
		t.Fatal(err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := wt.Add("CHANGELOG.md"); err != nil {
		t.Fatal(err)
	}

	if _, err := wt.Add(filepath.Join("bar", "CHANGELOG.md")); err != nil {
		t.Fatal(err)
	}

	if _, err := wt.Commit("release: missing module\n", &sgit.CommitOptions{
		Author: &object.Signature{
			Email: testutils.GotaggerEmail,
			Name:  testutils.GotaggerName,
			When:  time.Now(),
		},
	}); err != nil {
		t.Fatal(err)
	}

	g.Config.CreateTag = true
	_, err = g.TagRepo()
	assert.EqualError(t, err, "module validation failed:\nchanged modules not released by commit: foo/bar")
}

func TestGotagger_Version(t *testing.T) {
	g, repo, path, teardown := newGotagger(t)
	defer teardown()

	simpleGoRepo(t, repo, path)

	if v, err := g.Version(); assert.NoError(t, err) {
		assert.Equal(t, "v1.1.0", v)
	}
}

func TestGotagger_Version_no_module(t *testing.T) {
	g, repo, path, teardown := newGotagger(t)
	defer teardown()

	testutils.SimpleGitRepo(t, repo, path)

	if v, err := g.Version(); assert.NoError(t, err) {
		assert.Equal(t, "v1.1.0", v)
	}
}

func TestGotagger_Version_tag_head(t *testing.T) {
	g, repo, path, teardown := newGotagger(t)
	defer teardown()

	simpleGoRepo(t, repo, path)

	// tag HEAD higher than what gotagger would return
	version := "v1.10.0"
	testutils.CreateTag(t, repo, path, version)

	if got, err := g.Version(); assert.NoError(t, err) {
		assert.Equal(t, version, got)
	}
}

func TestGotagger_Version_PreMajor(t *testing.T) {
	g, repo, path, teardown := newGotagger(t)
	defer teardown()

	// set PreMajor
	g.Config.PreMajor = true

	simpleGoRepo(t, repo, path)

	// make a breaking change to foo
	testutils.CommitFile(t, repo, path, "foo.go", "feat!: breaking change", []byte(`contents`))

	// major version should rev
	if v, err := g.ModuleVersions("foo"); assert.NoError(t, err) {
		assert.Equal(t, []string{"v2.0.0"}, v)
	}

	// make a breaking change to sub/module
	testutils.CommitFile(t, repo, path, filepath.Join("sub", "module", "file"), "feat!: breaking change", []byte(`contents`))

	// version should not rev major
	if v, err := g.ModuleVersions("foo/sub/module"); assert.NoError(t, err) {
		assert.Equal(t, []string{"sub/module/v0.2.0"}, v)
	}
}

func TestGotagger_Version_breaking(t *testing.T) {
	g, repo, path, teardown := newGotagger(t)
	defer teardown()

	simpleGoRepo(t, repo, path)

	// make a breaking change
	testutils.CommitFile(t, repo, path, "new", "feat!: new is breaking", []byte("new data"))

	v, err := g.Version()
	if err != nil {
		t.Fatalf("Version() returned an error: %v", err)
	}

	if got, want := v, "v2.0.0"; got != want {
		t.Errorf("Version() returned %s, want %s", got, want)
	}
}

func TestNew(t *testing.T) {
	_, path, teardown := testutils.NewGitRepo(t)
	defer teardown()

	// invalid path should return an error
	_, err := New(filepath.FromSlash("/does/not/exist"))
	assert.Error(t, err)

	if g, err := New(path); assert.NoError(t, err) && assert.NotNil(t, g) {
		assert.Equal(t, NewDefaultConfig(), g.Config)
	}
}

func TestGotagger_findAllModules(t *testing.T) {
	tests := []struct {
		title    string
		repoFunc func(testutils.T, *sgit.Repository, string)
		include  []string
		exclude  []string
		want     []module
	}{
		{
			title:    "simple git repo",
			repoFunc: simpleGoRepo,
			want: []module{
				{".", "foo", ""},
				{filepath.Join("sub", "module"), "foo/sub/module", "sub/module/"},
			},
		},
		{
			title:    "v1 on master branch",
			repoFunc: masterV1GitRepo,
			want: []module{
				{".", "foo", ""},
				{"bar", "foo/bar", "bar/"},
			},
		},
		{
			title:    "v1 on master branch, exclude foo",
			repoFunc: masterV1GitRepo,
			exclude:  []string{"foo"},
			want: []module{
				{"bar", "foo/bar", "bar/"},
			},
		},
		{
			title:    "v1 on master branch, exclude all by path",
			repoFunc: masterV1GitRepo,
			exclude:  []string{"."},
		},
		{
			title:    "v1 on master branch, exclude foo/bar",
			repoFunc: masterV1GitRepo,
			exclude:  []string{"foo/bar"},
			want: []module{
				{".", "foo", ""},
			},
		},
		{
			title:    "v1 on master branch, exclude foo/bar by path",
			repoFunc: masterV1GitRepo,
			exclude:  []string{"bar"},
			want: []module{
				{".", "foo", ""},
			},
		},
		{
			title:    "v1 on master branch, include foo",
			repoFunc: masterV1GitRepo,
			include:  []string{"foo"},
			want: []module{
				{".", "foo", ""},
			},
		},
		{
			title:    "v1 on master branch, include foo/bar",
			repoFunc: masterV1GitRepo,
			include:  []string{"foo/bar"},
			want: []module{
				{"bar", "foo/bar", "bar/"},
			},
		},
		{
			title:    "v1 on master branch, explicitly include all",
			repoFunc: masterV1GitRepo,
			include:  []string{"foo", "foo/bar"},
			want: []module{
				{".", "foo", ""},
				{"bar", "foo/bar", "bar/"},
			},
		},
		{
			title:    "v1 on master branch, include none",
			repoFunc: masterV1GitRepo,
			include:  []string{"foz"},
		},
		{
			title:    "v2 on master branch",
			repoFunc: masterV2GitRepo,
			want: []module{
				{".", "foo/v2", ""},
				{"bar", "foo/bar/v2", "bar/"},
			},
		},
		{
			title:    "v2 directory",
			repoFunc: v2DirGitRepo,
			want: []module{
				{".", "foo", ""},
				{"v2", "foo/v2", ""},
				{"bar", "foo/bar", "bar/"},
				{filepath.Join("bar", "v2"), "foo/bar/v2", "bar/"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.title, func(t *testing.T) {
			t.Parallel()

			g, repo, path, teardown := newGotagger(t)
			defer teardown()

			tt.repoFunc(t, repo, path)

			g.Config.ExcludeModules = tt.exclude
			if modules, err := g.findAllModules(tt.include); assert.NoError(t, err) {
				assert.Equal(t, tt.want, modules)
			}
		})
	}
}

func Test_groupCommitsByModule(t *testing.T) {
	tests := []struct {
		title    string
		repoFunc func(testutils.T, *sgit.Repository, string)
		want     map[module][]string
	}{
		{
			title:    "simple git repo",
			repoFunc: simpleGoRepo,
			want: map[module][]string{
				{".", "foo", ""}: {
					"feat: add go.mod",
					"feat: bar\n\nThis is a great bar.",
					"feat: more foo",
					"feat: foo",
				},
				{filepath.Join("sub", "module"), "foo/sub/module", "sub/module/"}: {
					"fix: fix submodule",
					"feat: add a file to submodule",
					"feat: add a submodule",
				},
			},
		},
		{
			title:    "v1 on master branch",
			repoFunc: masterV1GitRepo,
			want: map[module][]string{
				{".", "foo", ""}: {
					"feat: add go.mod",
				},
				{"bar", "foo/bar", "bar/"}: {
					"feat: add bar/go.mod",
				},
			},
		},
		{
			title:    "v2 on master branch",
			repoFunc: masterV2GitRepo,
			want: map[module][]string{
				{".", "foo/v2", ""}: {
					"feat!: add foo/v2 go.mod",
					"feat: add go.mod",
				},
				{"bar", "foo/bar/v2", "bar/"}: {
					"feat!: add bar/v2 go.mod",
					"feat: add bar/go.mod",
				},
			},
		},
		{
			"v2 directory",
			v2DirGitRepo,
			map[module][]string{
				{".", "foo", ""}: {
					"feat: add go.mod",
				},
				{"v2", "foo/v2", ""}: {
					"feat!: add v2/go.mod",
				},
				{"bar", "foo/bar", "bar/"}: {
					"feat: add bar/go.mod",
				},
				{filepath.Join("bar", "v2"), "foo/bar/v2", "bar/"}: {
					"feat!: add bar/v2/go.mod",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			t.Parallel()

			g, repo, path, teardown := newGotagger(t)
			defer teardown()

			tt.repoFunc(t, repo, path)

			modules, err := g.findAllModules(nil)
			require.NoError(t, err)

			commits, err := g.repo.RevList("HEAD", "")
			require.NoError(t, err)

			groups := groupCommitsByModule(commits, modules)

			// groups is a map of module to commits, but we can't construct
			// commits so we convert to a map of modules to commit messages
			got := map[module][]string{}
			for module, commits := range groups {
				gotMessages := make([]string, len(commits))
				for i, c := range commits {
					gotMessages[i] = c.Message
				}
				got[module] = gotMessages
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGotagger_validateModules(t *testing.T) {
	tests := []struct {
		title   string
		commit  []module
		changed []module
		want    string
	}{
		{
			title:   "all match",
			commit:  []module{{".", "foo", ""}},
			changed: []module{{".", "foo", ""}},
			want:    "",
		},
		{
			title:   "extra bar",
			commit:  []module{{".", "foo", ""}, {"bar", "bar", "bar/"}},
			changed: []module{{".", "foo", ""}},
			want:    "module validation failed:\nmodules not changed by commit: bar",
		},
		{
			title:   "missing bar",
			commit:  []module{{".", "foo", ""}},
			changed: []module{{".", "foo", ""}, {"bar", "bar", "bar/"}},
			want:    "module validation failed:\nchanged modules not released by commit: bar",
		},
		{
			title:   "extra bar, baz",
			commit:  []module{{".", "foo", ""}, {"bar", "bar", "bar/"}, {"baz", "baz", "baz/"}},
			changed: []module{{".", "foo", ""}},
			want:    "module validation failed:\nmodules not changed by commit: bar, baz",
		},
		{
			title:   "missing bar, baz",
			commit:  []module{{".", "foo", ""}},
			changed: []module{{".", "foo", ""}, {"bar", "bar", "bar/"}, {"baz", "baz", "baz/"}},
			want:    "module validation failed:\nchanged modules not released by commit: bar, baz",
		},
		{
			title:   "extra bar, missing baz",
			commit:  []module{{".", "foo", ""}, {"bar", "bar", "bar/"}},
			changed: []module{{".", "foo", ""}, {"baz", "baz", "baz/"}},
			want:    "module validation failed:\nmodules not changed by commit: bar\nchanged modules not released by commit: baz",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.title, func(t *testing.T) {
			t.Parallel()

			err := validateCommitModules(tt.commit, tt.changed)
			if tt.want == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.want)
			}
		})
	}
}

func newGotagger(t testutils.T) (g *Gotagger, repo *sgit.Repository, path string, teardown func()) {
	t.Helper()

	repo, path, teardown = testutils.NewGitRepo(t)

	g = &Gotagger{
		Config: NewDefaultConfig(),
		repo: &git.Repository{
			Path: path,
			Repo: repo,
		},
	}

	return
}

// create a repo that has foo and foo/bar in master, and foo/v2 and foo/bar/v2 in v2.
func masterV1GitRepo(t testutils.T, repo *sgit.Repository, path string) {
	t.Helper()

	// setup v1 modules
	h := setupV1Modules(t, repo, path)

	// create a v2 branch
	b := plumbing.NewBranchReferenceName("v2")
	ref := plumbing.NewHashReference(b, h)
	if err := repo.Storer.SetReference(ref); err != nil {
		t.Fatal(err)
	}

	// v2 commits go into v2
	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	if err := w.Checkout(&sgit.CheckoutOptions{
		Branch: b,
	}); err != nil {
		t.Fatal(err)
	}

	setupV2Modules(t, repo, path)

	// checkout master
	if err := w.Checkout(&sgit.CheckoutOptions{
		Branch: plumbing.Master,
	}); err != nil {
		t.Fatal(err)
	}
}

// create a repo that has foo and foo/bar in v1, and foo/v2 and foo/bar/v2 in master.
func masterV2GitRepo(t testutils.T, repo *sgit.Repository, path string) {
	t.Helper()

	// create v1 modules
	h := setupV1Modules(t, repo, path)

	// create a v1 branch
	b := plumbing.NewBranchReferenceName("v1")
	ref := plumbing.NewHashReference(b, h)
	if err := repo.Storer.SetReference(ref); err != nil {
		t.Fatal(err)
	}

	// v2 commits go into master
	setupV2Modules(t, repo, path)
}

// create a repo with mixed tags
func mixedTagRepo(t testutils.T, repo *sgit.Repository, path string) {
	t.Helper()

	// create top-level go.mod and tag it v1.0.0
	testutils.CommitFile(t, repo, path, "go.mod", "feat: add go.mod", []byte("module foo\n"))
	testutils.CreateTag(t, repo, path, "v1.0.0")
	testutils.CommitFile(t, repo, path, "foo.go", "feat: add foo.go", []byte("foo\n"))

	// commit and tag it 0.1.0 (no prefix)
	testutils.CommitFile(t, repo, path, "bar.go", "feat: add bar.go", []byte("bar\n"))
	testutils.CreateTag(t, repo, path, "0.1.0")
}

func v2DirGitRepo(t testutils.T, repo *sgit.Repository, path string) {
	t.Helper()

	// create top-level go.mod and tag it v1.0.0
	testutils.CommitFile(t, repo, path, "go.mod", "feat: add go.mod", []byte("module foo\n"))
	testutils.CreateTag(t, repo, path, "v1.0.0")

	// create sub module and tag it v1.0.0
	testutils.CommitFile(t, repo, path, filepath.Join("bar", "go.mod"), "feat: add bar/go.mod", []byte("module foo/bar\n"))
	testutils.CreateTag(t, repo, path, "bar/v1.0.0")

	// create a v2 directory and tag v2.0.0
	testutils.CommitFile(t, repo, path, filepath.Join("v2", "go.mod"), "feat!: add v2/go.mod", []byte("module foo/v2\n"))
	testutils.CreateTag(t, repo, path, "v2.0.0")

	// create bar/v2 directory and tag bar/v2.0.0
	testutils.CommitFile(t, repo, path, filepath.Join("bar", "v2", "go.mod"), "feat!: add bar/v2/go.mod", []byte("module foo/bar/v2\n"))
	testutils.CreateTag(t, repo, path, "bar/v2.0.0")
}

func setupV1Modules(t testutils.T, repo *sgit.Repository, path string) (head plumbing.Hash) {
	t.Helper()

	// create top-level go.mod and tag it v1.0.0
	testutils.CommitFile(t, repo, path, "go.mod", "feat: add go.mod", []byte("module foo\n"))
	testutils.CreateTag(t, repo, path, "v1.0.0")

	// create sub module and tag it v1.0.0
	head = testutils.CommitFile(t, repo, path, filepath.Join("bar", "go.mod"), "feat: add bar/go.mod", []byte("module foo/bar\n"))
	testutils.CreateTag(t, repo, path, "bar/v1.0.0")

	return
}

func setupV2Modules(t testutils.T, repo *sgit.Repository, path string) (head plumbing.Hash) {
	t.Helper()

	testutils.CommitFile(t, repo, path, "go.mod", "feat!: add foo/v2 go.mod", []byte("module foo/v2\n"))
	testutils.CreateTag(t, repo, path, "v2.0.0")

	// update bar module to v2
	head = testutils.CommitFile(t, repo, path, filepath.Join("bar", "go.mod"), "feat!: add bar/v2 go.mod", []byte("module foo/bar/v2\n"))
	testutils.CreateTag(t, repo, path, "bar/v2.0.0")

	return
}

func simpleGoRepo(t testutils.T, repo *sgit.Repository, path string) {
	t.Helper()

	testutils.SimpleGitRepo(t, repo, path)
	testutils.CommitFile(t, repo, path, "go.mod", "feat: add go.mod", []byte("module foo\n"))
	testutils.CommitFile(t, repo, path, "sub/module/go.mod", "feat: add a submodule", []byte("module foo/sub/module\n"))
	testutils.CommitFile(t, repo, path, "sub/module/file", "feat: add a file to submodule", []byte("some data"))
	testutils.CreateTag(t, repo, path, "sub/module/v0.1.0")
	testutils.CommitFile(t, repo, path, "sub/module/file", "fix: fix submodule", []byte("some more data"))
}