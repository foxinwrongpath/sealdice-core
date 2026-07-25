package dice

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// 辅助函数（保持不变）
func spellRingsRenew(mctx *MsgContext, _ *Message) int {
	num := 0
	for i := 1; i < 10; i++ {
		ringMax, exists := VarGetValueInt64(mctx, fmt.Sprintf("$环位上限_%d", i))
		if exists {
			num++
			VarSetValueInt64(mctx, fmt.Sprintf("$环位_%d", i), ringMax)
		}
	}
	return num
}

func spellRingsGet(mctx *MsgContext, ring int64, num int64) (string, bool) {
	ringCur, _ := VarGetValueInt64(mctx, fmt.Sprintf("$环位_%d", ring))
	newRing := ringCur + num
	ringMax, _ := VarGetValueInt64(mctx, fmt.Sprintf("$环位上限_%d", ring))
	if newRing < 0 {
		return fmt.Sprintf("%s无法消耗%d个%d环环位，当前%d个", getPlayerNameTempFunc(mctx), -num, ring, ringCur), false
	}
	if newRing > ringMax {
		newRing = ringMax
	}
	VarSetValueInt64(mctx, fmt.Sprintf("$环位_%d", ring), newRing)
	if num < 0 {
		return fmt.Sprintf("%s的%d环环位消耗至%d个，上限%d个", getPlayerNameTempFunc(mctx), ring, newRing, ringMax), true
	}
	return fmt.Sprintf("%s的%d环环位恢复至%d个，上限%d个", getPlayerNameTempFunc(mctx), ring, newRing, ringMax), true
}

var cmdRing = &CmdItemInfo{
	Name:      "ring",
	ShortHelp: ".ring {show|init|set|use|rest|clear} ...",
	Help: "PMDnD 环位管理:\n" +
		".ring show                          查看当前环位状况\n" +
		".ring init <1环数> <2环数> ...      设置各环位上限（按顺序）\n" +
		".ring set <N环> <数量>              单独设置某环位上限\n" +
		".ring use <N环> [数量]              消耗N环环位（默认1个）\n" +
		".ring rest                          恢复所有环位\n" +
		".ring clear                         清除所有环位数据\n" +
		"示例：.ring init 4 3 2\n" +
		"      .ring set 2环 4\n" +
		"      .ring use 1\n" +
		"      .ring use 3 2",
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		// 兼容旧命令：如果第一个参数是数字或“环”，则自动转为 use
		first := cmdArgs.GetArgN(1)
		if matched, _ := regexp.MatchString(`^\d+[环cC]?$`, first); matched {
			cmdArgs.Args = append([]string{"ring", "use"}, cmdArgs.Args...)
		}
		cmdArgs.ChopPrefixToArgsWith("init", "set", "use", "rest", "clear", "show")
		sub := cmdArgs.GetArgN(1)
		mctx := GetCtxProxyFirst(ctx, cmdArgs)

		switch sub {
		case "show", "":
			var texts []string
			for i := 1; i < 10; i++ {
				ringCur, _ := VarGetValueInt64(mctx, fmt.Sprintf("$环位_%d", i))
				ringMax, exists := VarGetValueInt64(mctx, fmt.Sprintf("$环位上限_%d", i))
				if exists {
					texts = append(texts, fmt.Sprintf("%d环:%d/%d", i, ringCur, ringMax))
				}
			}
			summary := strings.Join(texts, ", ")
			if summary == "" {
				summary = "没有设置过环位"
			}
			ReplyToSender(mctx, msg, fmt.Sprintf("%s的环位状况: %s", getPlayerNameTempFunc(mctx), summary))

		case "init":
			reSlot := regexp.MustCompile(`\d+`)
			slots := reSlot.FindAllString(cmdArgs.CleanArgs, -1)
			if len(slots) > 0 {
				var texts []string
				for index, levelVal := range slots {
					v, _ := strconv.ParseInt(levelVal, 10, 64)
					VarSetValueInt64(mctx, fmt.Sprintf("$环位_%d", index+1), v)
					VarSetValueInt64(mctx, fmt.Sprintf("$环位上限_%d", index+1), v)
					texts = append(texts, fmt.Sprintf("%d环%d个", index+1, v))
				}
				ReplyToSender(mctx, msg, "为"+getPlayerNameTempFunc(mctx)+"设置环位: "+strings.Join(texts, ", "))
				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}
			} else {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

		case "set":
			reSlot := regexp.MustCompile(`(\d+)\s*[环cC]\s*(\d+)|[lL][vV](\d+)\s+(\d+)`)
			slots := reSlot.FindAllStringSubmatch(cmdArgs.CleanArgs, -1)
			if len(slots) > 0 {
				var texts []string
				for _, oneSlot := range slots {
					level := oneSlot[1]
					if level == "" {
						level = oneSlot[3]
					}
					levelVal := oneSlot[2]
					if levelVal == "" {
						levelVal = oneSlot[4]
					}
					iLevel, _ := strconv.ParseInt(level, 10, 64)
					iLevelVal, _ := strconv.ParseInt(levelVal, 10, 64)
					VarSetValueInt64(mctx, fmt.Sprintf("$环位_%d", iLevel), iLevelVal)
					VarSetValueInt64(mctx, fmt.Sprintf("$环位上限_%d", iLevel), iLevelVal)
					texts = append(texts, fmt.Sprintf("%d环%d个", iLevel, iLevelVal))
				}
				ReplyToSender(mctx, msg, "为"+getPlayerNameTempFunc(mctx)+"设置环位: "+strings.Join(texts, ", "))
			} else {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

		case "use":
			// 兼容旧 .cast 语法：直接数字或 环数
			rest := cmdArgs.CleanArgs
			reSlot := regexp.MustCompile(`(\d+)(?:[环cC]?|\s)\s*(\d+)?|[lL][vV](\d+)(?:\s+(\d+))?`)
			slots := reSlot.FindAllStringSubmatch(rest, -1)
			if len(slots) > 0 {
				for _, oneSlot := range slots {
					level := oneSlot[1]
					if level == "" {
						level = oneSlot[3]
					}
					levelVal := oneSlot[2]
					if levelVal == "" {
						levelVal = oneSlot[4]
					}
					if levelVal == "" {
						levelVal = "1"
					}
					iLevel, _ := strconv.ParseInt(level, 10, 64)
					iLevelVal, _ := strconv.ParseInt(levelVal, 10, 64)
					text, ok := spellRingsGet(mctx, iLevel, -iLevelVal)
					if !ok {
						ReplyToSender(mctx, msg, text)
						return CmdExecuteResult{Matched: true, Solved: true}
					}
					ReplyToSender(mctx, msg, text)
				}
			} else {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

		case "rest":
			n := spellRingsRenew(mctx, msg)
			if n > 0 {
				ReplyToSender(mctx, msg, fmt.Sprintf("%s的环位已经完全恢复", getPlayerNameTempFunc(mctx)))
			} else {
				ReplyToSender(mctx, msg, fmt.Sprintf("%s并没有设置过环位", getPlayerNameTempFunc(mctx)))
			}
			if ctx.Player.AutoSetNameTemplate != "" {
				_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
			}

		case "clear", "clr":
			attrs, err := mctx.Dice.AttrsManager.LoadByCtx(mctx)
			if err != nil {
				ReplyToSender(mctx, msg, "加载属性失败: "+err.Error())
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			for i := 1; i < 10; i++ {
				attrs.Delete(fmt.Sprintf("$环位_%d", i))
				attrs.Delete(fmt.Sprintf("$环位上限_%d", i))
			}
			ReplyToSender(mctx, msg, fmt.Sprintf("%s环位数据已清除", getPlayerNameTempFunc(mctx)))
			if ctx.Player.AutoSetNameTemplate != "" {
				_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
			}

		default:
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
