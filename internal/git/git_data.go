package git

import (
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Commit 单次提交
type Commit struct {
	Hash, Author, Date, Message string
}

// FileChange 文件变更统计
type FileChange struct {
	File    string
	Changes int
}

// Contributor 贡献者
type Contributor struct {
	Name  string
	Count int
}

// Info 聚合所有 Git 数据
type Info struct {
	Branches     []string
	Current      string
	Commits      []Commit
	Files        []FileChange
	Contributors []Contributor
	Ahead        int
	Behind       int
	Modified     int
	Added        int
	Deleted      int
	Untracked    int
	Err          error
}

// FetchInfoFromDir 执行 git 命令获取指定目录的仓库信息
func FetchInfoFromDir(dir string) Info {
	// git rev-parse --is-inside-work-tree: 检查是否在 git 工作树内
	if _, err := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree").Output(); err != nil {
		return Info{Err: err}
	}
	return fetchConcurrent(dir)
}

func fetchConcurrent(dir string) Info {
	info := &Info{}
	var wg sync.WaitGroup

	wg.Add(6)
	go func() {
		defer wg.Done()
		branches, current := branchesInDir(dir)
		info.Branches = branches
		info.Current = current
	}()
	go func() {
		defer wg.Done()
		info.Commits = commitsInDir(dir, 30)
	}()
	go func() {
		defer wg.Done()
		info.Files = fileChangesInDir(dir, 100)
	}()
	go func() {
		defer wg.Done()
		info.Contributors = contributorsInDir(dir, 100)
	}()
	go func() {
		defer wg.Done()
		ahead, behind := aheadBehindInDir(dir)
		info.Ahead = ahead
		info.Behind = behind
	}()
	go func() {
		defer wg.Done()
		modified, added, deleted, untracked := workingDirStatusInDir(dir)
		info.Modified = modified
		info.Added = added
		info.Deleted = deleted
		info.Untracked = untracked
	}()
	wg.Wait()
	return *info
}

func branchesInDir(dir string) ([]string, string) {
	// git branch --format=%(refname:short): 获取所有分支名（短格式）
	out, err := exec.Command("git", "-C", dir, "branch", "--format=%(refname:short)").Output()
	if err != nil {
		return nil, ""
	}
	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if s := strings.TrimSpace(line); s != "" {
			branches = append(branches, s)
		}
	}
	// git branch --show-current: 获取当前分支名
	cur, _ := exec.Command("git", "-C", dir, "branch", "--show-current").Output()
	return branches, strings.TrimSpace(string(cur))
}

func commitsInDir(dir string, n int) []Commit {
	// git log --oneline -n N --format=%h|%an|%ad|%s --date=short
	// 获取最近 N 次提交：哈希|作者|日期|消息
	out, err := exec.Command("git", "-C", dir, "log",
		"--oneline", "-n", strconv.Itoa(n),
		"--format=%h|%an|%ad|%s", "--date=short",
	).Output()
	if err != nil {
		return nil
	}
	var commits []Commit
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "|", 4)
		if len(parts) < 4 {
			continue
		}
		commits = append(commits, Commit{
			Hash:    parts[0],
			Author:  parts[1],
			Date:    parts[2],
			Message: parts[3],
		})
	}
	return commits
}

func fileChangesInDir(dir string, n int) []FileChange {
	// git log --name-only --pretty=format: -n N
	// 获取最近 N 次提交中修改的文件名，统计变更次数
	out, err := exec.Command("git", "-C", dir, "log", "--name-only", "--pretty=format:", "-n", strconv.Itoa(n)).Output()
	if err != nil {
		return nil
	}
	counts := map[string]int{}
	for _, line := range strings.Split(string(out), "\n") {
		if f := strings.TrimSpace(line); f != "" {
			counts[f]++
		}
	}
	var files []FileChange
	for f, c := range counts {
		files = append(files, FileChange{f, c})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Changes > files[j].Changes })
	if len(files) > 8 {
		files = files[:8]
	}
	return files
}

func contributorsInDir(dir string, n int) []Contributor {
	// git shortlog -sn --all -n N
	// 统计所有分支的贡献者及其提交次数，按次数降序排列
	out, err := exec.Command("git", "-C", dir, "shortlog", "-sn", "--all", "-n", strconv.Itoa(n)).Output()
	if err != nil {
		return nil
	}
	var contributors []Contributor
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			parts = strings.SplitN(line, "  ", 2)
		}
		if len(parts) < 2 {
			continue
		}
		count, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		contributors = append(contributors, Contributor{strings.TrimSpace(parts[1]), count})
	}
	return contributors
}

func aheadBehindInDir(dir string) (int, int) {
	// git rev-list --left-right --count HEAD...@{upstream}
	// 获取本地分支与上游分支的领先和落后次数
	out, err := exec.Command("git", "-C", dir, "rev-list", "--left-right", "--count", "HEAD...@{upstream}").Output()
	if err != nil {
		return 0, 0
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "\t")
	if len(parts) != 2 {
		return 0, 0
	}
	ahead, err1 := strconv.Atoi(parts[0])
	behind, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0
	}
	return ahead, behind
}

func workingDirStatusInDir(dir string) (int, int, int, int) {
	// git status --short: 获取暂存区和工作区的文件状态
	// X = 暂存区状态，Y = 工作区状态
	// M = modified, A = added, D = deleted, ?? = untracked
	out, err := exec.Command("git", "-C", dir, "status", "--short").Output()
	if err != nil {
		return 0, 0, 0, 0
	}
	modified := 0
	added := 0
	deleted := 0
	untracked := 0
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) < 3 {
			continue
		}
		status := line[:2]
		if status == "??" {
			untracked++
			continue
		}
		// 统计 M/A/D（检查暂存区X和工作区Y两个位置）
		if strings.ContainsAny(status, "M") {
			modified++
		} else if strings.ContainsAny(status, "A") {
			added++
		} else if strings.ContainsAny(status, "D") {
			deleted++
		}
	}
	return modified, added, deleted, untracked
}
