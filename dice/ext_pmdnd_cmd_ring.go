package dice

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

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
	Name:      "环位",
	ShortHelp: ".环位 // 查看当前环位状况\n.ring // 同.环位\n.环位 init 4 3 2 // 设置1 2 3环的环位上限\n.环位 set 2环 4 // 单独设置某一环的环位上限\n.环位 clr // 清除环位设置\n.环位 rest // 恢复所有环位\n.环位 3环 +1 // 增加一个3环环位\n.环位 3环 -1 // 消耗一个3环环位",
	Help: "PMDnD 环位(.环位 .ring):\n" +
		".环位 // 查看当前环位状况\n.ring // 同.环位\n.环位 init 4 3 2 // 设置1 2 3环的环位上限\n.环位 set 2环 4 // 单独设置某一环的环位上限\n.环位 clr // 清除环位设置\n.环位 rest // 恢复所有环位\n.环位 3环 +1 // 增加一个3环环位\n.环位 3环 -1 // 消耗一个3环环位",
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		cmdArgs.ChopPrefixToArgsWith("init", "set")
		val := cmdArgs.GetArgN(1)
		mctx := GetCtxProxyFirst(ctx, cmdArgs)

		switch val {
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
		case "clr":
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
		case "":
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
		case "help":
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		default:
			reSlot := regexp.MustCompile(`(\d+)\s*[环cC]\s*([+-])\s*(\d+)|[lL][vV](\d+)\s*([+-])\s*(\d+)`)
			slots := reSlot.FindAllStringSubmatch(cmdArgs.CleanArgs, -1)
			if len(slots) > 0 {
				for _, oneSlot := range slots {
					level := oneSlot[1]
					if level == "" {
						level = oneSlot[4]
					}
					op := oneSlot[2]
					if op == "" {
						op = oneSlot[5]
					}
					levelVal := oneSlot[3]
					if levelVal == "" {
						levelVal = oneSlot[6]
					}
					iLevel, _ := strconv.ParseInt(level, 10, 64)
					iLevelVal, _ := strconv.ParseInt(levelVal, 10, 64)
					if op == "-" {
						iLevelVal = -iLevelVal
					}
					text, ok := spellRingsGet(mctx, iLevel, iLevelVal)
					if !ok {
						ReplyToSender(mctx, msg, text)
						return CmdExecuteResult{Matched: true, Solved: true}
					}
					ReplyToSender(mctx, msg, text)
				}
			} else {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}

var cmdCast = &CmdItemInfo{
	Name:          "cast",
	ShortHelp:     ".cast 1 // 消耗1个1环环位\n.cast 1 2 // 消耗2个1环环位",
	Help:          "PMDnD 使用技能(.cast):\n.cast 1 // 消耗1个1环环位\n.cast 1 2 // 消耗2个1环环位",
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		mctx := GetCtxProxyFirst(ctx, cmdArgs)
		val := cmdArgs.GetArgN(1)
		switch val {
		case "help", "":
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		case "环位":
			val2 := cmdArgs.GetArgN(2)
			if val2 == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			ring, err := strconv.ParseInt(val2, 10, 64)
			if err != nil {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			text, ok := spellRingsGet(mctx, ring, -1)
			ReplyToSender(mctx, msg, text)
			if !ok {
				return CmdExecuteResult{Matched: true, Solved: true}
			}
		default:
			reSlot := regexp.MustCompile(`(\d+)(?:[环cC]|\s)\s*(\d+)?|[lL][vV](\d+)(?:\s+(\d+))?`)
			slots := reSlot.FindAllStringSubmatch(cmdArgs.CleanArgs, -1)
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
					ReplyToSender(mctx, msg, text)
					if !ok {
						return CmdExecuteResult{Matched: true, Solved: true}
					}
				}
			} else {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}

var cmdLongRest = &CmdItemInfo{
	Name:          "长休",
	ShortHelp:     ".长休 // 恢复生命值和环位\n.longrest // 另一种写法",
	Help:          "PMDnD 休息:\n.长休 // 恢复生命值和环位\n.longrest // 另一种写法",
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		mctx := GetCtxProxyFirst(ctx, cmdArgs)
		tmpl := mctx.Group.GetCharTemplate(mctx.Dice)
		if tmpl != nil {
			mctx.SystemTemplate = tmpl
		}
		isShort := cmdArgs.Command == "短休" || cmdArgs.Command == "shortrest" || cmdArgs.Command == "dshortrest"
		recoveryRate := int64(1)
		restType := "长休"
		if isShort {
			recoveryRate = 2
			restType = "短休"
		}

		hpText := "没有设置hpmax，无法回复hp"
		hpMax, exists := VarGetValueInt64(mctx, "hpmax")
		if exists {
			curHp, _ := VarGetValueInt64(mctx, "hp")
			recoveredHP := hpMax / recoveryRate
			if isShort && curHp > recoveredHP && curHp <= hpMax {
				recoveredHP = curHp
			}
			if recoveryRate == 1 {
				recoveredHP = hpMax
			}
			VarSetValueInt64(mctx, "hp", recoveredHP)
			hpText = fmt.Sprintf("hp恢复至%d", recoveredHP)
		}

		n := spellRingsRenew(mctx, msg)
		ringText := ""
		if n > 0 {
			ringText = "，环位得到了恢复"
		}
		if ctx.Player.AutoSetNameTemplate != "" {
			_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
		}
		ReplyToSender(mctx, msg, fmt.Sprintf("%s的%s: %s%s", getPlayerNameTempFunc(mctx), restType, hpText, ringText))
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}

var cmdShortRest = cmdLongRest // 短休复用长休，但命令名称不同会区分
