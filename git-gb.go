package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	ioutil "io/ioutil"

	git "github.com/libgit2/git2go"
	"github.com/mgutz/ansi"
)

var (
	Red    string = ansi.ColorCode("red")
	Yellow        = ansi.ColorCode("yellow")
	Green         = ansi.ColorCode("green")
)

const (
	BaseBranch string = "master"

	CachePath = ".git/go_gb_cache.json"
)

func exit(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	fmt.Println(msg)
	os.Exit(1)
}

func NewRepo() *git.Repository {
	repo, err := git.OpenRepository(".")
	if err != nil {
		wd, _ := os.Getwd()
		exit("Could not open repository at '%s'", wd)
	}
	return repo
}

func NewBranchIterator(repo *git.Repository) *git.BranchIterator {
	i, err := repo.NewBranchIterator(git.BranchLocal)
	if err != nil {
		wd, _ := os.Getwd()
		exit("Failed to list branches for '%s'", wd)
	}
	return i
}

func LookupBaseOid(repo *git.Repository) *git.Oid {
	base_branch, err := repo.LookupBranch(BaseBranch, git.BranchLocal)
	if err != nil {
		exit("Error looking up '%s'", BaseBranch)
	}

	return base_branch.Target()
}

type Comparison struct {
	Repo    *git.Repository
	BaseOid *git.Oid
	Branch  *git.Branch
	Oid     *git.Oid

	IsMerged bool
	Ahead    int
	Behind   int
}

func NewComparison(repo *git.Repository, base_oid *git.Oid, branch *git.Branch, store CacheStore) *Comparison {
	c := new(Comparison)

	c.Repo = repo
	c.BaseOid = base_oid

	c.Branch = branch
	c.Oid = branch.Target()

	cache := store[c.CacheKey()]

	if cache != nil {
		c.Ahead = cache.Ahead
		c.Behind = cache.Behind
		c.IsMerged = cache.IsMerged
	} else {
		c.IsMerged = false
		c.Ahead = -1
		c.Behind = -1
	}

	return c
}

func (c *Comparison) Name() string {
	name, err := c.Branch.Name()
	if err != nil {
		exit("Can't get branch name for '%s'", c.Oid)
	}
	return name
}

func (c *Comparison) IsHead() bool {
	head, err := c.Branch.IsHead()
	if err != nil {
		exit("Can't get IsHead for '%s'", c.Name())
	}
	return head
}

func (c *Comparison) Commit() *git.Commit {
	commit, err := c.Repo.LookupCommit(c.Oid)
	if err != nil {
		exit("Could not lookup commit '%s'.", c.Oid.String())
	}
	return commit
}

func (c *Comparison) ColorCode() string {
	hours, _ := time.ParseDuration("336h") // two weeks
	two_weeks := time.Now().Add(-hours)

	if c.IsHead() {
		return Green
	} else if c.When().Before(two_weeks) {
		return Red
	} else {
		return Yellow
	}
}

func (c *Comparison) When() time.Time {
	sig := c.Commit().Committer()
	return sig.When
}

func (c *Comparison) FormattedWhen() string {
	return c.When().Format("2006-01-02 15:04PM")
}

func (c *Comparison) CacheKey() string {
	strs := []string{c.BaseOid.String(), c.Oid.String()}
	return strings.Join(strs, "..")
}

func (c *Comparison) SetIsMerged() {
	if c.Oid.String() == c.BaseOid.String() {
		c.IsMerged = true
	} else {
		merged, err := c.Repo.DescendantOf(c.BaseOid, c.Oid)
		if err != nil {
			exit("Could not get descendant of '%s' and '%s'.", c.BaseOid.String(), c.Oid.String())
		}
		c.IsMerged = merged
	}
}

func (c *Comparison) SetAheadBehind() {
	var err error
	c.Ahead, c.Behind, err = c.Repo.AheadBehind(c.Oid, c.BaseOid)
	if err != nil {
		exit("Error getting ahead/behind", c.BaseOid.String())
	}
}

func (c *Comparison) Execute() {
	if c.Ahead > -1 && c.Behind > -1 {
		return
	}

	c.SetIsMerged()
	c.SetAheadBehind()
}

type Comparisons []*Comparison

func (cs Comparisons) MaxBranchLength() int {
	max := 30

	for _, comp := range cs {
		length := utf8.RuneCountInString(comp.Name())
		if length > max {
			max = length
		}
	}
	return max
}

type ComparisonsByWhen Comparisons

func (a ComparisonsByWhen) Len() int {
	return len(a)
}

func (a ComparisonsByWhen) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ComparisonsByWhen) Less(i, j int) bool {
	return a[i].When().Unix() < a[j].When().Unix()
}

type Options struct {
	Ahead      int
	Behind     int
	Merged     bool
	NoMerged   bool
	ClearCache bool
}

func NewOptions() *Options {
	o := new(Options)

	flag.IntVar(&o.Ahead, "ahead", -1, "only show branches that are <ahead> commits ahead.")
	flag.IntVar(&o.Behind, "behind", -1, "only show branches that are <behind> commits behind.")
	flag.BoolVar(&o.Merged, "merged", false, "only show branches that are merged.")
	flag.BoolVar(&o.NoMerged, "no-merged", false, "only show branches that are not merged.")
	flag.BoolVar(&o.ClearCache, "clear-cache", false, "clear cache of comparisons.")

	flag.Parse()

	return o
}

type CacheStore map[string]*Comparison

func NewCacheStore() CacheStore {
	bits, err := ioutil.ReadFile(CachePath)
	if err != nil {
		// no-op: `cache.json` will be written on exit.
	}

	y := make(CacheStore)
	_ = json.Unmarshal(bits, &y)

	return y
}

func (store *CacheStore) WriteToFile() error {
	b, err := json.Marshal(store)
	if err != nil {
		fmt.Printf("Could not save cache to file.")
	}
	ioutil.WriteFile(CachePath, b, 0644)
	return nil
}

func main() {
	opts := NewOptions()

	if opts.ClearCache {
		os.Remove(CachePath)
	}

	store := NewCacheStore()

	repo := NewRepo()
	branch_iterator := NewBranchIterator(repo)
	base_oid := LookupBaseOid(repo)

	comparisons := make(Comparisons, 0)

	// type BranchIteratorFunc func(*Branch, BranchType) error
	branch_iterator.ForEach(func(branch *git.Branch, btype git.BranchType) error {
		comp := NewComparison(repo, base_oid, branch, store)
		comparisons = append(comparisons, comp)
		return nil
	})

	sort.Sort(ComparisonsByWhen(comparisons))

	branch_length := comparisons.MaxBranchLength()

	for _, comp := range comparisons {
		comp.Execute()

		merged_string := ""
		if comp.IsMerged {
			merged_string = "(merged)"
		}

		if opts.Ahead != -1 && opts.Ahead != comp.Ahead {
			continue
		}

		if opts.Behind != -1 && opts.Behind != comp.Behind {
			continue
		}

		if opts.Merged && !comp.IsMerged {
			continue
		}

		if opts.NoMerged && comp.IsMerged {
			continue
		}

		fmt.Printf(
			"%s%s | %-*s | behind: %4d | ahead: %4d %s\n",
			comp.ColorCode(),
			comp.FormattedWhen(),
			branch_length, // http://stackoverflow.com/a/28870241
			comp.Name(),
			comp.Behind,
			comp.Ahead,
			merged_string)

		store[comp.CacheKey()] = comp
	}

	store.WriteToFile()

}
