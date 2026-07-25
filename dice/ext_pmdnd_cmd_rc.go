package dice

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	ds "github.com/sealdice/dicescript"
)

var cmdRc = &CmdItemInfo{
	EnableExecuteTimesParse: true,
	Name:                    "rc",
	ShortHelp:               ".rc [--hide] [优势/劣势] <表达式> [@目标]",
	Help: "PMDnD 技能检定:\n" +
		".rc <表达式>              普通检定\n" +
		".rc 优势 <表达式>         优势检定\n" +
		".rc 劣势 <表达式>         劣势检定\n" +
		".rc --hide <表达式>       暗骰（结果私聊）\n" +
		".rc <表达式> @某人        为他人检定（需授权）\n" +
		"示例：.rc 力量+3\n" +
		"      .rc 优势 力量+3\n" +
		"      .rc --hide 感知+1",
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		mctx := GetCtxProxyFirst(ctx, cmdArgs)
		if ctx != nil {
			mctx.DelegateText = ctx.DelegateText
			if ctx.Dice != nil {
				if baseTmpl, ok := ctx.Dice.GameSystemMap.Load("pmdnd"); ok && baseTmpl != nil {
					mctx.SystemTemplate = baseTmpl
				}
			}
		}
		// 检查 --hide
		hide := false
		clean := cmdArgs.CleanArgs
		if strings.Contains(clean, "--hide") {
			hide = true
			clean = strings.Replace(clean, "--hide", "", 1)
			cmdArgs.CleanArgs = clean
		}

		val := cmdArgs.GetArgN(1)
		switch val {
		case "", "help":
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		default:
			restText := cmdArgs.CleanArgs
			re := regexp.MustCompile(`^(优势|劣势|優勢|劣勢)`)
			m := re.FindString(restText)
			if m != "" {
				m = strings.Replace(m, "優勢", "优势", 1)
				m = strings.Replace(m, "劣勢", "劣势", 1)
				restText = strings.TrimSpace(restText[len(m):])
			}
			mctx.CreateVmIfNotExists()
			tmpl := mctx.Group.GetCharTemplate(mctx.Dice)
			if tmpl != nil {
				mctx.SystemTemplate = tmpl
			}
			textList := make([]string, 0)
			round := 1
			if cmdArgs.SpecialExecuteTimes > 1 {
				round = cmdArgs.SpecialExecuteTimes
			}
			if cmdArgs.SpecialExecuteTimes > int(ctx.Dice.Config.MaxExecuteTime) && cmdArgs.SpecialExecuteTimes != 1 {
				VarSetValueStr(mctx, "$t次数", strconv.Itoa(cmdArgs.SpecialExecuteTimes))
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "DND:检定_轮数过多警告"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			for range round {
				mctx.Eval(tmpl.InitScript, nil)
				mctx.setDndReadForVM(true)
				expr := fmt.Sprintf("d20%s", m)
				r := mctx.Eval(expr, nil)
				if r.vm.Error != nil {
					ReplyToSender(mctx, msg, "无法解析表达式: "+restText)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				d20Result, _ := r.ReadInt()
				VarSetValueInt64(mctx, "$t骰子出目", int64(d20Result))
				diceDetail := r.vm.GetDetailText()
				if diceDetail == "" || diceDetail == strconv.Itoa(int(d20Result)) {
					diceDetail = fmt.Sprintf("%d[d20]", d20Result)
				}
				expr = restText
				r2 := mctx.Eval(expr, nil)
				if r2.vm.Error != nil {
					ReplyToSender(mctx, msg, "无法解析表达式: "+restText)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				reason := r2.vm.RestInput
				if reason == "" {
					reason = restText
				}
				reason = LimitCommandReasonText(reason)
				modifier, ok := r2.ReadInt()
				if !ok {
					if r2.TypeId == ds.VMTypeFloat {
						modifier = ds.IntType(math.Floor(r2.MustReadFloat()))
					} else {
						ReplyToSender(mctx, msg, fmt.Sprintf("属性非数字类型，无法用于检定: %s", restText))
						return CmdExecuteResult{Matched: true, Solved: true}
					}
				}
				modifierDetail := r2.vm.GetDetailText()
				if modifierDetail == "" {
					modifierDetail = r2.ToString()
				}
				detail := fmt.Sprintf("%s + %s", diceDetail, modifierDetail)
				VarSetValueStr(mctx, "$t技能", reason)
				VarSetValueStr(mctx, "$t检定过程文本", detail)
				VarSetValueInt64(mctx, "$t检定结果", int64(d20Result+modifier))
				if round == 1 {
					textList = append(textList, DiceFormatTmpl(mctx, "DND:检定"))
				} else {
					textList = append(textList, DiceFormatTmpl(mctx, "DND:检定_单项结果文本"))
				}
			}
			var text string
			if round > 1 {
				VarSetValueStr(mctx, "$t结果文本", strings.Join(textList, "\n"))
				VarSetValueStr(mctx, "$t次数", strconv.Itoa(cmdArgs.SpecialExecuteTimes))
				text = DiceFormatTmpl(mctx, "DND:检定_多轮")
			} else {
				text = textList[0]
			}

			// --- 宝可梦化：添加风格前缀 ---
			playerName := getPlayerNameTempFunc(mctx)
			if round > 1 {
				text = "🎯 " + playerName + " 的多次检定结果：\n" + text
			} else {
				text = "🎯 " + playerName + " 的检定！\n" + text
			}

			if hide {
				if ctx.Group != nil && !ctx.IsPrivate {
					ctx.CommandHideFlag = ctx.Group.GroupID
					ReplyGroup(ctx, msg, DiceFormatTmpl(mctx, "核心:暗骰_群内"))
					ReplyPerson(ctx, msg, text)
				} else {
					ReplyToSender(ctx, msg, text)
				}
			} else {
				ReplyToSender(mctx, msg, text)
			}
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
