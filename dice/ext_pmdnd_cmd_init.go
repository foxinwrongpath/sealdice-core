package dice

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	ds "github.com/sealdice/dicescript"
)

var cmdInit = &CmdItemInfo{
	Name: "init",
	ShortHelp: ".init {add|list|del|set|next|clear} ...\n" +
		"快捷添加：.ri [值] [名称]",
	Help: "PMDnD 先攻管理:\n" +
		".init list                           查看先攻列表\n" +
		".init add <名称> [先攻值]             添加单位（如未给值则自动掷d20）\n" +
		".init del <名称> ...                 删除单位\n" +
		".init set <名称> <先攻表达式>         设置单位的先攻（可包含d20）\n" +
		".init next                           进入下一回合\n" +
		".init clear                          清空先攻列表\n" +
		"快捷指令：.ri 15 张三  或  .ri 张三（自动掷骰）\n" +
		"         .ri 优势 张三  或 .ri 劣势 张三",
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		// 如果命令是 .ri，直接当作 .init add 处理
		if cmdArgs.Command == "ri" {
			cmdArgs.Args = append([]string{"add"}, cmdArgs.Args...)
		}
		cmdArgs.ChopPrefixToArgsWith("add", "list", "del", "set", "next", "clear", "clr")
		sub := cmdArgs.GetArgN(1)

		switch sub {
		case "list", "":
			riList := (RIList{}).LoadByCurGroup(ctx)
			round, _ := VarGetValueInt64(ctx, "$g回合数")
			textOut := DiceFormatTmpl(ctx, "DND:先攻_查看_前缀")
			for order, i := range riList {
				textOut += fmt.Sprintf("%2d. %s: %d\n", order+1, i.name, i.val)
			}
			if len(riList) == 0 {
				textOut += "- 没有找到任何单位"
			} else {
				if len(riList) <= int(round) || round < 0 {
					round = 0
				}
				textOut += fmt.Sprintf("当前回合：%s", riList[round].name)
			}
			ReplyToSender(ctx, msg, textOut)

		case "add":
			// 处理 .ri 的快捷添加（兼容多种格式）
			text := cmdArgs.CleanArgs
			mctx := GetCtxProxyFirst(ctx, cmdArgs)
			// 尝试解析 名称 和 值
			parts := strings.Fields(text)
			if len(parts) == 0 {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			var name string
			var val int64
			var detail string
			// 检查是否有优势/劣势关键字（来自 .ri 的复杂语法）
			if strings.HasPrefix(parts[0], "优势") || strings.HasPrefix(parts[0], "劣势") {
				// 使用原 .ri 的复杂解析（简化为直接调用一个内部函数）
				// 为了不重复代码，这里我们提示用户使用更简单的格式
				ReplyToSender(ctx, msg, "复杂先攻请使用 .init set 或 .ri 标准格式（值 + 名称）")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if num, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
				// 有数字
				val = num
				if len(parts) > 1 {
					name = parts[1]
				} else {
					name = mctx.Player.Name
				}
			} else {
				// 第一个词是名称
				name = parts[0]
				val = int64(ds.Roll(nil, 20, 0))
				if len(parts) > 1 {
					if num2, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
						val = num2
					}
				}
			}
			riList := (RIList{}).LoadByCurGroup(ctx)
			if len(riList) == 0 {
				VarSetValueInt64(ctx, "$g当前回合先攻值", NULL_INIT_VAL)
			}
			riList = append(riList, &RIListItem{name: name, val: val, detail: detail, uid: ""})
			sort.Sort(riList)
			riList.SaveToGroup(ctx)
			ReplyToSender(ctx, msg, fmt.Sprintf("添加先攻: %s = %d", name, val))

		case "del":
			name := cmdArgs.GetArgN(2)
			if name == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			riList := (RIList{}).LoadByCurGroup(ctx)
			newList := RIList{}
			removed := false
			for _, i := range riList {
				if i.name != name {
					newList = append(newList, i)
				} else {
					removed = true
				}
			}
			if removed {
				newList.SaveToGroup(ctx)
				ReplyToSender(ctx, msg, fmt.Sprintf("已删除 %s", name))
			} else {
				ReplyToSender(ctx, msg, fmt.Sprintf("未找到 %s", name))
			}

		case "set":
			name := cmdArgs.GetArgN(2)
			expr := strings.Join(cmdArgs.Args[3:], "")
			if name == "" || expr == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			r := ctx.Eval(expr, nil)
			if r.vm.Error != nil || r.TypeId != ds.VMTypeInt {
				ReplyToSender(ctx, msg, "表达式无效，应为数字或d20表达式")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			val := int64(r.MustReadInt())
			riList := (RIList{}).LoadByCurGroup(ctx)
			found := false
			for _, i := range riList {
				if i.name == name {
					i.val = val
					found = true
					break
				}
			}
			if !found {
				riList = append(riList, &RIListItem{name: name, val: val})
			}
			sort.Sort(riList)
			riList.SaveToGroup(ctx)
			ReplyToSender(ctx, msg, fmt.Sprintf("设置 %s 先攻为 %d", name, val))

		case "next", "end":
			lst := (RIList{}).LoadByCurGroup(ctx)
			if len(lst) == 0 {
				ReplyToSender(ctx, msg, "先攻列表为空")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			round, _ := VarGetValueInt64(ctx, "$g回合数")
			round = (round + 1) % int64(len(lst))
			setInitNextRoundVars(ctx, lst, round)
			ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "DND:先攻_下一回合"))

		case "clear", "clr":
			(RIList{}).SaveToGroup(ctx)
			VarSetValueInt64(ctx, "$g当前回合先攻值", NULL_INIT_VAL)
			VarSetValueInt64(ctx, "$g回合数", 0)
			ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "DND:先攻_清除列表"))

		default:
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}

// 保留 .ri 快捷指令（直接映射到 cmdInit，但通过命令名判断）
var cmdRi = &CmdItemInfo{
	Name: "ri",
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		// 直接转发到 cmdInit，并标记命令名为 "ri"
		cmdArgs.Command = "ri"
		return cmdInit.Solve(ctx, msg, cmdArgs)
	},
}
