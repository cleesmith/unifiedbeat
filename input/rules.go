package input

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/elastic/libbeat/logp"
)

type Rule struct {
	SourceFileIndex   int
	SourceFileLineNum int
	Gid               string
	Sid               string
	Msg               string
	RuleRaw           string
}

var SourceFiles []string

var Rules = make(map[string]Rule)

func LoadRules(genMsgMapPath string, rulePaths []string) (int, int, error) {
	// this func should only be called once during unifiedbeat initialization
	multipleLineWarnings := 0
	duplicateRuleWarnings := 0

	duplicateRuleWarnings, err := loadGenMsgMap(genMsgMapPath)

	backslash := `\` // indicates a multiple line snort rule to be ignored

	defaultGid := "1"

	ruleActionsRegexp := `^alert|^log|^pass|^activate|^dynamic|^drop|^reject|^sdrop`
	ruleSidRegexp := `sid\s*:\s*(\d+);`
	ruleGidRegexp := `gid\s*:\s*(\d+);`
	ruleMsgRegexp := `msg\s*:\s*\"(.*?)\";`

	matchRuleActions, err := regexp.Compile(ruleActionsRegexp)
	if err != nil {
		fmt.Println(err)
		return 0, 0, err
	}

	matchRuleSid, err := regexp.Compile(ruleSidRegexp)
	if err != nil {
		fmt.Println(err)
		return 0, 0, err
	}

	matchRuleGid, err := regexp.Compile(ruleGidRegexp)
	if err != nil {
		fmt.Println(err)
		return 0, 0, err
	}

	matchRuleMsg, err := regexp.Compile(ruleMsgRegexp)
	if err != nil {
		fmt.Println(err)
		return 0, 0, err
	}

	// create a list of files based on rulePaths array (unifiedbeat.rules.paths in unifiedbeat.yml):
	var ruleFileNames []string
	for _, apath := range rulePaths {
		// evaluate apath as a wildcards/shell glob
		matches, err := filepath.Glob(apath)
		if err != nil {
			logp.Debug("rules", "filepath.Glob(%s) failed: %v", apath, err)
			return 0, 0, err
		}
		for _, amatch := range matches {
			logp.Debug("rules", "processing matched file: %s", amatch)
			// stat the file, following any symlinks
			fileinfo, err := os.Stat(amatch)
			if err != nil {
				logp.Debug("rules", "os.Stat(%s) failed: %s", amatch, err)
				continue
			}
			if fileinfo.IsDir() {
				dir, err := os.Open(amatch) // open folder to get list of rules files
				if err != nil {
					fmt.Println(err)
					return 0, 0, err
				}
				fileNames, err := dir.Readdirnames(-1)
				if err != nil {
					fmt.Println(err)
					return 0, 0, err
				}
				dir.Close()
				for _, aFileName := range fileNames {
					ruleFileNames = append(ruleFileNames, path.Join(dir.Name(), aFileName))
				}
			} else {
				ruleFileNames = append(ruleFileNames, amatch)
			}
		}
	}

	// process each rule file:
	sourceFileIndex := 0
	for _, filename := range ruleFileNames {
		aFile, err := os.Open(filename)
		if err != nil {
			fmt.Println(err)
			return 0, 0, err
		}
		// avoid duplicating path and filename's for each rule (less memory)
		SourceFiles = append(SourceFiles, aFile.Name())
		sourceFileIndex = len(SourceFiles) - 1

		scanner := bufio.NewScanner(aFile)
		lineNum := 0
		for scanner.Scan() {
			aline := strings.TrimSpace(scanner.Text())
			lineNum++
			if len(aline) <= 0 {
				continue
			}
			// at a minimum, a Rule must consist of an "action", "sid", and "msg"
			matchedRuleAction := matchRuleActions.MatchString(aline)
			if matchedRuleAction {
				// no regexp's seem to match a backlash "\" at the end of a line,
				// so just check the last character instead:
				eol := len(aline) - 1
				lastChar := string(aline[eol])
				if lastChar == backslash {
					// maybe, some day, deal with multi-line rules, maybe
					fmt.Printf("WARNING ignoring \"multiple line\" Rule on line# %v from file:\n\t%v\n", lineNum, aFile.Name())
					multipleLineWarnings++
					continue
				}
				// check for "gid:?;", but usually rules default to gid=1 see:
				// http://manual.snort.org/node31.html#SECTION00443000000000000000
				matchedRuleGid := matchRuleGid.FindStringSubmatch(aline)
				if len(matchedRuleGid) < 2 {
					matchedRuleGid = append(matchedRuleGid, "")
					matchedRuleGid = append(matchedRuleGid, defaultGid)
				}
				matchedRuleSid := matchRuleSid.FindStringSubmatch(aline)
				if len(matchedRuleSid) < 2 {
					continue
				}
				gid_sid := matchedRuleGid[1] + ":" + matchedRuleSid[1]

				matchedRuleMsg := matchRuleMsg.FindStringSubmatch(aline)
				if len(matchedRuleMsg) < 2 {
					continue
				}

				// this line is a rule, so add it to Rules unless it's a duplicate
				inUseRule, isDuplicateRule := Rules[gid_sid]
				if isDuplicateRule {
					// first rule found wins, who knows how Snort handles this issue
					fmt.Printf("\nWARNING ignoring \"duplicate\" Rule on line# %v from file:\n\t%v\n", lineNum, aFile.Name())
					fmt.Printf("\tduplicate of Rule on line# %v from file:\n", inUseRule.SourceFileLineNum)
					fmt.Printf("\t%v\n", SourceFiles[inUseRule.SourceFileIndex])
					fmt.Printf("\tgid_sid=%v\n", gid_sid)
					// debug duplicate rules:
					// shellcode.rules often has duplicate rules based on gid+sid but with different protocols (tcp vs udp)
					duplicateRuleWarnings++
					continue
				} else {
					Rules[gid_sid] = Rule{sourceFileIndex, lineNum, matchedRuleGid[1], matchedRuleSid[1], matchedRuleMsg[1], aline}
				}
			}
		}
		aFile.Close()
	}
	return multipleLineWarnings, duplicateRuleWarnings, err
}

func loadGenMsgMap(genMsgMapPath string) (int, error) {
	var duplicateRuleWarnings int
	f, err := os.Open(genMsgMapPath)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}
	defer f.Close()
	SourceFiles = append(SourceFiles, genMsgMapPath)
	sourceFileIndex := len(SourceFiles) - 1

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		aline := scanner.Text()
		lineNum++
		words := strings.Split(aline, "||")
		if len(words) >= 3 {
			gid := strings.TrimSpace(words[0])
			sid := strings.TrimSpace(words[1])
			msg := strings.TrimSpace(words[2])
			gid_sid := gid + ":" + sid
			inUseRule, isDuplicateRule := Rules[gid_sid]
			if isDuplicateRule {
				// first rule found wins, who knows how Snort handles this issue
				fmt.Printf("WARNING ignoring \"duplicate\" Rule on line# %v from file:\n\t%v\n", lineNum, genMsgMapPath)
				fmt.Printf("\tduplicate of Rule on line# %v from file:\n", inUseRule.SourceFileLineNum)
				fmt.Printf("\t%v\n", SourceFiles[inUseRule.SourceFileIndex])
				duplicateRuleWarnings++
				continue
			} else {
				Rules[gid_sid] = Rule{sourceFileIndex, lineNum, gid, sid, msg, aline}
			}
		}
	}
	return duplicateRuleWarnings, err
}
