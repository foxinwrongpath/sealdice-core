package dice

import (
	"fmt"
	"strconv"
)

var cmdRing = &CmdItemInfo{
	Name:      "ring",
	ShortHelp: ".ring show              查看最大可用环位\n.ring set <环位>        设置最大可用环位",
	Help:          getRingHelp(),
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		sub := cmdArgs.GetArgN(1)
		mctx := GetCtxProxyFirst(ctx, cmdArgs)

		switch sub {
		case "show", "":
			maxRing, exists := VarGetValueInt64(mctx, "$max_ring")
			if !exists {
				ReplyToSender(mctx, msg, "未设置最大环位，默认为 0（无法使用需要环位的招式）")
			} else {
				ReplyToSender(mctx, msg, fmt.Sprintf("最大可用环位: %d环", maxRing))
			}

		case "set":
			valStr := cmdArgs.GetArgN(2)
			if valStr == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			ringVal, err := strconv.ParseInt(valStr, 10, 64)
			if err != nil || ringVal < 0 {
				ReplyToSender(mctx, msg, "请输入有效的非负整数")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if ringVal > 9 {
				ringVal = 9
			}
			VarSetValueInt64(mctx, "$max_ring", ringVal)
			ReplyToSender(mctx, msg, fmt.Sprintf("最大可用环位已设置为: %d环", ringVal))

		default:
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
