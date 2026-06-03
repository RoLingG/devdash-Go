// ============================================================================
// git.go — Git 命令调用 + 数据解析
// ============================================================================

package main

import (
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// GitCommit 单次提交
type GitCommit struct {
	Hash, Author, Date, Message string
}

// GitFileChange 文件变更统计
type GitFileChange struct {
	File    string
	Changes int
}

// GitContributor 贡献者
type GitContributor struct {
	Name  string
	Count int
}

// GitInfo 聚合所有 Git 数据
type GitInfo struct {
	Branches     []string
	Current      string
	Commits      []GitCommit
	Files        []GitFileChange
	Contributors []GitContributor
	Err          error
}

// fetchGitInfoFromDir 执行 git 命令获取指定目录的仓库信息
func fetchGitInfoFromDir(dir string) GitInfo {
	if _, err := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree").Output(); err != nil {
		return GitInfo{Err: err}
	}
	info := GitInfo{}
	info.Branches, info.Current = gitBranchesInDir(dir)
	info.Commits = gitCommitsInDir(dir, 30)
	info.Files = gitFileChangesInDir(dir, 100)
	info.Contributors = gitContributorsInDir(dir, 100)
	return info
}

// fetchGitInfo 执行 git 命令获取当前目录的仓库信息（保持向后兼容）
func fetchGitInfo() GitInfo {
	return fetchGitInfoFromDir(".")
}

// gitBranchesInDir 获取指定目录的分支列表和当前分支
func gitBranchesInDir(dir string) ([]string, string) {
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

// gitBranches 获取当前目录的分支列表（保持向后兼容）
func gitBranches() ([]string, string) {
	return gitBranchesInDir(".")
}

// gitCommitsInDir 获取指定目录的最近 n 次提交
func gitCommitsInDir(dir string, n int) []GitCommit {
	out, err := exec.Command("git", "-C", dir, "log",
		"--oneline", "-n", strconv.Itoa(n),
		"--format=%h|%an|%ad|%s", "--date=short",
	).Output()
	if err != nil {
		return nil
	}
	var commits []GitCommit
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "|", 4)
		if len(parts) < 4 {
			continue
		}
		commits = append(commits, GitCommit{
			Hash: parts[0], Author: parts[1], Date: parts[2], Message: parts[3],
		})
	}
	return commits
}

// gitCommits 获取当前目录的最近 n 次提交（保持向后兼容）
func gitCommits(n int) []GitCommit {
	return gitCommitsInDir(".", n)
}

// gitFileChangesInDir 统计指定目录变更最频繁的文件
func gitFileChangesInDir(dir string, n int) []GitFileChange {
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
	var files []GitFileChange
	for f, c := range counts {
		files = append(files, GitFileChange{f, c})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Changes > files[j].Changes })
	if len(files) > 8 {
		files = files[:8]
	}
	return files
}

// gitFileChanges 统计当前目录变更最频繁的文件（保持向后兼容）
func gitFileChanges(n int) []GitFileChange {
	return gitFileChangesInDir(".", n)
}

// gitContributorsInDir 统计指定目录的贡献者排行
func gitContributorsInDir(dir string, n int) []GitContributor {
	out, err := exec.Command("git", "-C", dir, "shortlog", "-sn", "--all", "-n", strconv.Itoa(n)).Output()
	if err != nil {
		return nil
	}
	var contribs []GitContributor
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 格式: "  42  Author Name"（tab 或双空格分隔）
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
		contribs = append(contribs, GitContributor{strings.TrimSpace(parts[1]), count})
	}
	return contribs
}

// gitContributors 统计当前目录的贡献者排行（保持向后兼容）
func gitContributors(n int) []GitContributor {
	return gitContributorsInDir(".", n)
}
