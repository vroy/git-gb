package main

import(
	"time"
	"os"
	"fmt"
	"github.com/libgit2/git2go"
)

const (
	Red    string = "\x1b[0;31m"
	Yellow string = "\x1b[0;33m"
	Green  string = "\x1b[0;32m"

	BaseBranch string = "master"
)


func GetRepo() (*git.Repository) {
	repo, err := git.OpenRepository(".")
	if err != nil {
		// @todo improve message
		fmt.Printf("Could not open repository at '.'\n")
		os.Exit(1)
	}
	return repo
}

func GetBranchIterator(repo *git.Repository) (*git.BranchIterator) {
	i, err := repo.NewBranchIterator(git.BranchLocal)
	if err != nil {
		// @todo improve message
		fmt.Printf("Can't list branches\n")
		os.Exit(1)
	}
	return i
}

func GetBaseOid(repo *git.Repository) (*git.Oid) {
	base_branch, err := repo.LookupBranch(BaseBranch, git.BranchLocal)
	if err != nil {
		fmt.Printf("Error looking up %s\n", BaseBranch)
		os.Exit(1)
	}

	return base_branch.Target()
}


type Comparison struct {
	Repo *git.Repository
	BaseOid *git.Oid
	Branch *git.Branch
	Oid *git.Oid

	_ahead int
	_behind int
}

func NewComparison(repo *git.Repository, base_oid *git.Oid, branch *git.Branch) *Comparison{
	c := new(Comparison)

	c.Repo = repo
	c.BaseOid = base_oid

	c.Branch = branch
	c.Oid = branch.Target()

	c._ahead = -1
	c._behind = -1

	return c
}

func (c Comparison) Name() string {
	name, err := c.Branch.Name()
	if err != nil {
		fmt.Printf("Can't get branch name\n")
		os.Exit(1)
	}
	return name
}

func (c Comparison) IsHead() bool {
	head, err := c.Branch.IsHead()
	if err != nil {
		fmt.Printf("Can't get IsHead\n")
		os.Exit(1)
	}
	return head
}


func (c Comparison) IsMerged() bool {
		if c.Oid.String() == c.BaseOid.String() {
			return true
		} else {
			merged, err := c.Repo.DescendantOf(c.BaseOid, c.Oid)
			if err != nil {
				fmt.Printf("Could not get descendant of '%s' and '%s'.\n", c.BaseOid, c.Oid)
				os.Exit(1)
			}
			return merged
		}
}

func (c Comparison) Commit() *git.Commit {
	commit, err := c.Repo.LookupCommit(c.Oid)
	if err != nil {
		fmt.Printf("Could not lookup commit '%s'.\n", c.Oid)
		os.Exit(1)
	}
	return commit
}

// @todo red for old commits
func (c Comparison) Color() string {
	if c.IsHead() {
		return Green
	} else {
		return Yellow
	}
}

func (c Comparison) When() time.Time {
	sig := c.Commit().Committer()
	return sig.When
}

func (c *Comparison) Ahead() int {
	c.ComputeAheadBehind()
	return c._ahead
}

func (c *Comparison) Behind() int {
	c.ComputeAheadBehind()
	return c._behind
}

func (c *Comparison) ComputeAheadBehind() {
	if (c._ahead > -1 && c._behind > -1) { return }

	var err error
	c._ahead, c._behind, err = c.Repo.AheadBehind(c.Oid, c.BaseOid)
	if err != nil {
		fmt.Printf("Error getting ahead/behind\n", c.BaseOid)
		os.Exit(1)
	}
}


func main() {
	repo := GetRepo()
	branch_iterator := GetBranchIterator(repo)
	base_oid := GetBaseOid(repo)

	// comparisons := [...]Comparison


	// type BranchIteratorFunc func(*Branch, BranchType) error
	branch_iterator.ForEach( func(branch *git.Branch, btype git.BranchType) error {
		comp := NewComparison(repo, base_oid, branch)

		merged_string := ""
		if comp.IsMerged() {
			merged_string = "(merged)"
		}

		fmt.Printf(
			"%s%s | %-30s           | behind: %4d | ahead: %4d %s\n",
			comp.Color(),
			comp.When().Format("2006-01-02 15:04"),
			comp.Name(),
			comp.Behind(),
			comp.Ahead(),
			merged_string)

		return nil
	})

	// @todo store all comparisons in a list that can be sorted before printing.
	// @todo filter them things
}
