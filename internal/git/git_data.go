package git

import (
	"os/exec"
	"sort"
	"strconv"
	"strings"
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
	Err          error
}

// FetchInfoFromDir 执行 git 命令获取指定目录的仓库信息
func FetchInfoFromDir(dir string) Info {
	if _, err := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree").Output(); err != nil {
		return Info{Err: err}
	}
	info := Info{}
	info.Branches, info.Current = branchesInDir(dir)
	info.Commits = commitsInDir(dir, 30)
	info.Files = fileChangesInDir(dir, 100)
	info.Contributors = contributorsInDir(dir, 100)
	info.Ahead, info.Behind = aheadBehindInDir(dir)
	return info
}

func branchesInDir(dir string) ([]string, string) {
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
	cur, _ := exec.Command("git", "-C", dir, "branch", "--show-current").Output()
	return branches, strings.TrimSpace(string(cur))
}

func commitsInDir(dir string, n int) []Commit {
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
	out, err := exec.Command("git", "-C", dir, "shortlog", "-sn", "--all", "-n", strconv.Itoa(n)).Output()
	if err != nil {
		return nil
	}
	var contribs []Contributor
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
		contribs = append(contribs, Contributor{strings.TrimSpace(parts[1]), count})
	}
	return contribs
}

func aheadBehindInDir(dir string) (int, int) {
	// git rev-list --left-right --count HEAD...@{upstream}
	// 输出格式: "3\t2" (ahead \t behind)
	out, err := exec.Command("git", "-C", dir, "rev-list", "--left-right", "--count", "HEAD...@{upstream}").Output()
	if err != nil {
		// 没有配置 upstream 或其他错误，返回 0, 0
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
