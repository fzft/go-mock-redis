package main

var (
	gitSHA1   string = "unknown"
	gitDirty  string = "unknown"
	buildID   string = "unknown"
	buildDate string = "unknown"
)

func RedisGitSHA1() string {
	return gitSHA1
}

func RedisGitDirty() string {
	return gitDirty
}

func RedisBuildIdRaw() string {
	return buildID + buildDate + gitSHA1 + gitDirty
}
