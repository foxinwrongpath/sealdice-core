package dice

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	ds "github.com/sealdice/dicescript"
)

var cmdPmdnd = &CmdItemInfo{
	Name:      "pmdnd",
	ShortHelp: ".pmdnd [<数量>] // 制卡指令，返回<数量>组人物属性，最高10次\n.pmdndx [<数量>] // 制卡指令，带属性名，最高10次",
	Help: "PMDnD制卡指令:\n" +
		".pmdnd [<数量>] // 制卡指令，返回<数量>组人物属性，最高10次\n.pmdndx [<数量>] // 制卡指令，带属性名，最高10次",
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		isMode2 := cmdArgs.Command == "pmdndx"
		n := cmdArgs.GetArgN(1)
		val, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			if n == "" {
				val = 1
			} else {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
		}
		if val > 10 {
			val = 10
		}
		var ss []string
		for i := int64(0); i < val; i++ {
			if isMode2 {
				r := ctx.EvalFString(`力量:{$t1=4d6k3} 体质:{$t2=4d6k3} 敏捷:{$t3=4d6k3} 智力:{$t4=4d6k3} 感知:{$t5=4d6k3} 魅力:{$t6=4d6k3} 共计:{$tT=$t1+$t2+$t3+$t4+$t5+$t6}`, nil)
				if r.vm.Error != nil {
					break
				}
				result := r.ToString() + "\n"
				ss = append(ss, result)
			} else {
				r := ctx.EvalFString(`{4d6k3}, {4d6k3}, {4d6k3}, {4d6k3}, {4d6k3}, {4d6k3}`, nil)
				if r.vm.Error != nil {
					break
				}
				result := r.ToString()
				var nums Int64SliceDesc
				total := int64(0)
				for _, i := range strings.Split(result, ", ") {
					val, _ := strconv.ParseInt(i, 10, 64)
					nums = append(nums, val)
					total += val
				}
				sort.Sort(nums)
				var items []string
				for _, i := range nums {
					items = append(items, strconv.FormatInt(i, 10))
				}
				ret := fmt.Sprintf("[%s] = %d\n", strings.Join(items, ", "), total)
				ss = append(ss, ret)
			}
		}
		sep := DiceFormatTmpl(ctx, "DND:制卡_分隔符")
		info := strings.Join(ss, sep)
		VarSetValueStr(ctx, "$t制卡结果文本", info)
		var text string
		if isMode2 {
			text = DiceFormatTmpl(ctx, "DND:制卡_预设模式")
		} else {
			text = DiceFormatTmpl(ctx, "DND:制卡_自由分配模式")
		}
		ReplyToSender(ctx, msg, text)
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}

var cmdRi = &CmdItemInfo{
	Name:          "ri",
	ShortHelp:     ".ri 小明 // 值为D20\n.ri 12 张三 // 值12\n.ri +2 李四 // 值为D20+2\n.ri =D10+3 王五 // 值为D10+3\n.ri 优势 张三, 劣势-1 李四 // 支持优势劣势",
	Help:          "先攻设置:\n.ri 小明 // 值为D20\n.ri 12 张三 // 值12\n.ri +2 李四 // 值为D20+2\n.ri =D10+3 王五 // 值为D10+3\n.ri 优势 张三, 劣势-1 李四 // 支持优势劣势",
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		text := cmdArgs.CleanArgs
		mctx := GetCtxProxyFirst(ctx, cmdArgs)

		if cmdArgs.IsArgEqual(1, "help") {
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		}

		readOne := func() (int, string, int64, string, string) {
			text = strings.TrimSpace(text)
			var name string
			var val int64
			var detail string
			var exprExists bool
			var uid string

			if strings.HasPrefix(text, "+") {
				r := mctx.Eval("d20"+text, nil)
				if r.vm.Error != nil {
					return 1, "", 0, "", ""
				}
				detail = r.vm.GetDetailText()
				val = int64(r.MustReadInt())
				text = r.vm.RestInput
				exprExists = true
			} else if strings.HasPrefix(text, "-") {
				r := mctx.Eval("d20"+text, nil)
				if r.vm.Error != nil {
					return 1, "", 0, "", ""
				}
				detail = r.vm.GetDetailText()
				val = int64(r.MustReadInt())
				text = r.vm.RestInput
				exprExists = true
			} else if strings.HasPrefix(text, "=") {
				r := mctx.Eval(text[1:], nil)
				if r.vm.Error != nil {
					return 1, "", 0, "", ""
				}
				val = int64(r.MustReadInt())
				detail = r.vm.GetDetailText()
				text = r.vm.RestInput
				exprExists = true
			} else if strings.HasPrefix(text, "优势") || strings.HasPrefix(text, "劣势") {
				r := mctx.Eval("d20"+text, nil)
				if r.vm.Error != nil {
					return 2, "", 0, "", ""
				}
				detail = r.vm.GetDetailText()
				val = int64(r.MustReadInt())
				text = r.vm.RestInput
				exprExists = true
			} else {
				reNum := regexp.MustCompile(`^(\d+)`)
				m := reNum.FindStringSubmatch(text)
				if len(m) > 0 {
					val, _ = strconv.ParseInt(m[0], 10, 64)
					text = text[len(m[0]):]
					exprExists = true
				}
			}

			text = strings.TrimSpace(text)
			if strings.HasPrefix(text, ",") || strings.HasPrefix(text, "，") || text == "" {
				text = strings.TrimPrefix(text, ",")
				text = strings.TrimPrefix(text, "，")
				name = mctx.Player.Name
				name = strings.ReplaceAll(name, " ", "_")
				name = strings.ReplaceAll(name, "\n", "_")
				if !exprExists {
					val = int64(ds.Roll(nil, 20, 0))
				}
				uid = mctx.Player.UserID
				return 0, name, val, detail, uid
			}

			reName := regexp.MustCompile(`^([^\s\d,，][^\s,，]*)\s*[,，]?`)
			m := reName.FindStringSubmatch(text)
			if len(m) > 0 {
				name = m[1]
				text = text[len(m[0]):]
				if !exprExists {
					val = int64(ds.Roll(nil, 20, 0))
				}
			} else {
				return 2, "", 0, "", ""
			}
			return 0, name, val, detail, ""
		}

		solved := true
		tryOnce := true
		var items RIList

		for tryOnce || text != "" {
			code, name, val, detail, uid := readOne()
			items = append(items, &RIListItem{name, val, detail, uid})

			if code != 0 {
				solved = false
				break
			}
			tryOnce = false
		}

		if solved {
			riList := (RIList{}).LoadByCurGroup(ctx)

			var textOut strings.Builder
			textOut.WriteString(DiceFormatTmpl(mctx, "DND:先攻_设置_前缀"))
			sort.Sort(items)
			if riList.Len() == 0 {
				VarSetValueInt64(ctx, "$g当前回合先攻值", NULL_INIT_VAL)
			}
			for order, i := range items {
				var detail string
				if i.detail != "" {
					detail = i.detail + "="
				}
				_, _ = fmt.Fprintf(&textOut, "%2d. %s: %s%d\n", order+1, i.name, detail, i.val)

				item := riList.GetExists(i.name)
				if item == nil {
					curInitVal, _ := VarGetValueInt64(ctx, "$g当前回合先攻值")
					if i.val > curInitVal {
						round, _ := VarGetValueInt64(ctx, "$g回合数")
						VarSetValueInt64(ctx, "$g回合数", round+1)
					}
					riList = append(riList, i)
				} else {
					item.val = i.val
				}
			}

			sort.Sort(riList)
			riList.SaveToGroup(ctx)
			ReplyToSender(ctx, msg, textOut.String())
		} else {
			ReplyToSender(ctx, msg, DiceFormatTmpl(mctx, "DND:先攻_设置_格式错误"))
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}

var cmdInit = &CmdItemInfo{
	Name: "init",
	ShortHelp: ".init // 查看先攻列表\n" +
		".init del <单位1> <单位2> ... // 从先攻列表中删除\n" +
		".init set <单位名称> <先攻表达式> // 设置单位的先攻\n" +
		".init clr // 清除先攻列表\n" +
		".init end // 结束一回合\n" +
		".init help // 显示帮助",
	Help: ".init // 查看先攻列表\n" +
		".init del <单位1> <单位2> ... // 从先攻列表中删除\n" +
		".init set <单位名称> <先攻表达式> // 设置单位的先攻\n" +
		".init clr // 清除先攻列表\n" +
		".init end // 结束一回合\n" +
		".init help // 显示帮助",
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		cmdArgs.ChopPrefixToArgsWith("del", "set", "rm", "ed")
		n := cmdArgs.GetArgN(1)
		switch n {
		case "", "list":
			var textOut strings.Builder
			textOut.WriteString(DiceFormatTmpl(ctx, "DND:先攻_查看_前缀"))
			riList := (RIList{}).LoadByCurGroup(ctx)

			round, _ := VarGetValueInt64(ctx, "$g回合数")

			for order, i := range riList {
				_, _ = fmt.Fprintf(&textOut, "%2d. %s: %d\n", order+1, i.name, i.val)
			}

			if len(riList) == 0 {
				textOut.WriteString("- 没有找到任何单位")
			} else {
				if len(riList) <= int(round) || round < 0 {
					round = 0
				}
				rounder := riList[round]
				_, _ = fmt.Fprintf(&textOut, "当前回合：%s", rounder.name)
			}

			ReplyToSender(ctx, msg, textOut.String())
		case "ed", "end":
			lst := (RIList{}).LoadByCurGroup(ctx)
			round, _ := VarGetValueInt64(ctx, "$g回合数")
			if len(lst) == 0 {
				ReplyToSender(ctx, msg, "先攻列表为空")
				break
			}
			round = (round + 1) % int64(len(lst))

			setInitNextRoundVars(ctx, lst, round)
			ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "DND:先攻_下一回合"))
		case "del", "rm":
			tryDeleteMembersInInitList := func(deleteNames []string, riList RIList) (newList RIList, textOut strings.Builder, ok bool) {
				if len(riList) == 0 {
					textOut.WriteString("- 没有找到任何单位[先攻列表为空]\n")
					return riList, textOut, false
				}
				round, _ := VarGetValueInt64(ctx, "$g回合数")
				round %= int64(len(riList))
				toDeleted := map[string]bool{}
				for _, i := range deleteNames {
					toDeleted[i] = true
				}

				delCounter := 0

				preCurrent := 0
				for index, i := range riList {
					if !toDeleted[i.name] {
						newList = append(newList, i)
					} else {
						delCounter++
						_, _ = fmt.Fprintf(&textOut, "%2d. %s\n", delCounter, i.name)

						if int64(index) < round {
							preCurrent++
						}
					}
				}
				current := *riList[round]
				currentDeleted := toDeleted[current.name]

				round -= int64(preCurrent)
				if round >= int64(len(newList)) {
					round = 0
				}
				VarSetValueInt64(ctx, "$g回合数", round)

				if delCounter == 0 {
					textOut.WriteString("- 没有找到任何单位\n")
					return newList, textOut, false
				}

				newList.SaveToGroup(ctx)
				if currentDeleted {
					if len(newList) == 0 {
						VarSetValueInt64(ctx, "$g当前回合先攻值", NULL_INIT_VAL)
						textOut.WriteString(DiceFormatTmpl(ctx, "DND:先攻_清除列表"))
					} else {
						setInitNextRoundVars(ctx, newList, round)
						VarSetValueStr(ctx, "$t当前回合角色名", current.name)
						VarSetValueStr(ctx, "$t当前回合at", AtBuild(current.uid))
						textOut.WriteString(DiceFormatTmpl(ctx, "DND:先攻_下一回合"))
					}
				}
				return newList, textOut, true
			}

			nameWithSpace, _ := cmdArgs.EatPrefixWith("del", "rm")
			riList := (RIList{}).LoadByCurGroup(ctx)
			_, textOut, ok := tryDeleteMembersInInitList([]string{nameWithSpace}, riList)
			if !ok {
				_, textOut, _ = tryDeleteMembersInInitList(cmdArgs.Args[1:], riList)
			}
			textToSend := DiceFormatTmpl(ctx, "DND:先攻_移除_前缀") + textOut.String()

			ReplyToSender(ctx, msg, textToSend)
		case "set":
			name := cmdArgs.GetArgN(2)
			exists := name != ""
			arg3 := cmdArgs.GetArgN(3)
			exists2 := arg3 != ""
			if !exists || !exists2 {
				ReplyToSender(ctx, msg, "错误的格式，应为: .init set <单位名称> <先攻表达式>")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			expr := strings.Join(cmdArgs.Args[2:], "")
			r := ctx.Eval(expr, nil)
			if r.vm.Error != nil || r.TypeId != ds.VMTypeInt {
				ReplyToSender(ctx, msg, "错误的格式，应为: .init set <单位名称> <先攻表达式>")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			riList := (RIList{}).LoadByCurGroup(ctx)
			added := false
			for _, i := range riList {
				if i.name == name {
					i.val = int64(r.MustReadInt())
					added = true
					break
				}
			}
			if !added {
				if len(riList) == 0 {
					VarSetValueInt64(ctx, "$g当前回合先攻值", NULL_INIT_VAL)
				} else {
					curInitVal, _ := VarGetValueInt64(ctx, "$g当前回合先攻值")
					if int64(r.MustReadInt()) > curInitVal {
						round, _ := VarGetValueInt64(ctx, "$g回合数")
						VarSetValueInt64(ctx, "$g回合数", round+1)
					}
				}
				riList = append(riList, &RIListItem{name, int64(r.MustReadInt()), "", ""})
			}
			sort.Sort(riList)

			VarSetValueStr(ctx, "$t表达式", expr)
			VarSetValueStr(ctx, "$t目标", name)
			VarSetValueStr(ctx, "$t计算过程", r.vm.GetDetailText())
			VarSetValue(ctx, "$t点数", &r.VMValue)
			textOut := DiceFormatTmpl(ctx, "DND:先攻_设置_指定单位")

			riList.SaveToGroup(ctx)
			ReplyToSender(ctx, msg, textOut)
		case "clr", "clear":
			(RIList{}).SaveToGroup(ctx)
			VarSetValueInt64(ctx, "$g当前回合先攻值", NULL_INIT_VAL)
			ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "DND:先攻_清除列表"))
			VarSetValueInt64(ctx, "$g回合数", 0)
		case "help":
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		}

		return CmdExecuteResult{Matched: true, Solved: true}
	},
}

var cmdAction = &CmdItemInfo{
	Name:      "行动",
	ShortHelp: ".行动 // 查看当前行动资源\n.action // 同.行动\n.行动 用 // 消耗1个行动力\n.行动 ba // 消耗1个附加行动\n.行动 re // 消耗1个反应\n.行动 恢复 // 恢复全部行动资源\n.行动 set 行动力 2 // 设置行动力上限",
	Help: "PMDnD 行动经济(.行动 .action):\n" +
		".行动 // 查看当前行动资源\n.action // 同.行动\n.行动 用 // 消耗1个行动力\n.行动 ba // 消耗1个附加行动\n.行动 re // 消耗1个反应\n.行动 恢复 // 恢复全部行动资源\n.行动 set 行动力 2 // 设置行动力上限",
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		val := cmdArgs.GetArgN(1)
		mctx := GetCtxProxyFirst(ctx, cmdArgs)

		switch val {
		case "", "list", "show":
			action, _ := VarGetValueInt64(mctx, "行动力")
			if action == 0 {
				action = 1
			}
			bonus, _ := VarGetValueInt64(mctx, "附加行动")
			if bonus == 0 {
				bonus = 1
			}
			reac, _ := VarGetValueInt64(mctx, "反应")
			if reac == 0 {
				reac = 1
			}
			actionCur, _ := VarGetValueInt64(mctx, "$行动力_cur")
			bonusCur, _ := VarGetValueInt64(mctx, "$附加行动_cur")
			reacCur, _ := VarGetValueInt64(mctx, "$反应_cur")

			ReplyToSender(mctx, msg, fmt.Sprintf("%s的行动资源: 行动力 %d/%d  附加行动 %d/%d  反应 %d/%d",
				getPlayerNameTempFunc(mctx), actionCur, action, bonusCur, bonus, reacCur, reac))

		case "用", "use":
			actionCur, _ := VarGetValueInt64(mctx, "$行动力_cur")
			if actionCur <= 0 {
				if action, _ := VarGetValueInt64(mctx, "行动力"); action != 0 {
					actionCur = action
				} else {
					actionCur = 1
				}
			}
			if actionCur <= 0 {
				ReplyToSender(mctx, msg, fmt.Sprintf("%s没有可用的行动力", getPlayerNameTempFunc(mctx)))
			} else {
				VarSetValueInt64(mctx, "$行动力_cur", actionCur-1)
				ReplyToSender(mctx, msg, fmt.Sprintf("%s使用了行动力(%d→%d)", getPlayerNameTempFunc(mctx), actionCur, actionCur-1))
			}

		case "ba", "bonus", "附加":
			bonusCur, _ := VarGetValueInt64(mctx, "$附加行动_cur")
			if bonusCur <= 0 {
				if bonus, _ := VarGetValueInt64(mctx, "附加行动"); bonus != 0 {
					bonusCur = bonus
				} else {
					bonusCur = 1
				}
			}
			if bonusCur <= 0 {
				ReplyToSender(mctx, msg, fmt.Sprintf("%s没有可用的附加行动", getPlayerNameTempFunc(mctx)))
			} else {
				VarSetValueInt64(mctx, "$附加行动_cur", bonusCur-1)
				ReplyToSender(mctx, msg, fmt.Sprintf("%s使用了附加行动(%d→%d)", getPlayerNameTempFunc(mctx), bonusCur, bonusCur-1))
			}

		case "re", "react", "反应":
			reacCur, _ := VarGetValueInt64(mctx, "$反应_cur")
			if reacCur <= 0 {
				if reac, _ := VarGetValueInt64(mctx, "反应"); reac != 0 {
					reacCur = reac
				} else {
					reacCur = 1
				}
			}
			if reacCur <= 0 {
				ReplyToSender(mctx, msg, fmt.Sprintf("%s没有可用的反应", getPlayerNameTempFunc(mctx)))
			} else {
				VarSetValueInt64(mctx, "$反应_cur", reacCur-1)
				ReplyToSender(mctx, msg, fmt.Sprintf("%s使用了反应(%d→%d)", getPlayerNameTempFunc(mctx), reacCur, reacCur-1))
			}

		case "恢复", "rest", "reset":
			action, _ := VarGetValueInt64(mctx, "行动力")
			if action == 0 {
				action = 1
			}
			bonus, _ := VarGetValueInt64(mctx, "附加行动")
			if bonus == 0 {
				bonus = 1
			}
			reac, _ := VarGetValueInt64(mctx, "反应")
			if reac == 0 {
				reac = 1
			}
			VarSetValueInt64(mctx, "$行动力_cur", action)
			VarSetValueInt64(mctx, "$附加行动_cur", bonus)
			VarSetValueInt64(mctx, "$反应_cur", reac)
			ReplyToSender(mctx, msg, fmt.Sprintf("%s恢复了全部行动资源", getPlayerNameTempFunc(mctx)))

		case "set":
			target := cmdArgs.GetArgN(2)
			valStr := cmdArgs.GetArgN(3)
			if target == "" || valStr == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			n, err := strconv.ParseInt(valStr, 10, 64)
			if err != nil {
				ReplyToSender(mctx, msg, "数值必须是数字")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			switch target {
			case "行动力", "action":
				VarSetValueInt64(mctx, "行动力", n)
			case "附加行动", "bonusAction", "ba":
				VarSetValueInt64(mctx, "附加行动", n)
			case "反应", "reaction", "re":
				VarSetValueInt64(mctx, "反应", n)
			default:
				ReplyToSender(mctx, msg, "请指定: 行动力/附加行动/反应")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			ReplyToSender(mctx, msg, fmt.Sprintf("已设置%s为%d", target, n))

		case "help":
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		default:
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
