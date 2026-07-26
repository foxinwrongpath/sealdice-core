package dice

import (
	"strings"
)

// cmdRc 复用 D&D5E 的 RC 检定核心逻辑。
// PMDnD 仅新增 --hide 语法糖和宝可梦风格前缀。
// d20 骰点、优势/劣势解析、多轮检定、模板变量设置等全部委托给 dnd5eCmdRc.Solve。
var cmdRc = &CmdItemInfo{
	EnableExecuteTimesParse: true,
	Name:                    "rc",
	ShortHelp:               ".rc [--hide] [优势/劣势] <表达式> [@目标]",
	Help:   getRcHelp(),
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		// PMDnD 专属: --hide 语法糖，转换为 D&D5E 兼容的 rah/rch 暗骰命令
		if strings.Contains(cmdArgs.CleanArgs, "--hide") {
			cmdArgs.CleanArgs = strings.Replace(cmdArgs.CleanArgs, "--hide", "", 1)
			cmdArgs.Command = "rah"
		}

		// 设置 PMDnD 模板到上下文（dnd5eCmdRc 内部会加载群组模板覆盖）
		mctx := GetCtxProxyFirst(ctx, cmdArgs)
		if ctx != nil {
			mctx.DelegateText = ctx.DelegateText
		}
		if ctx.Dice != nil {
			if pmdndTmpl, ok := ctx.Dice.GameSystemMap.Load("pmdnd"); ok && pmdndTmpl != nil {
				mctx.SystemTemplate = pmdndTmpl
			}
		}

		// 委托给 D&D5E RC 核心执行（已导出为 dnd5eCmdRc）
		return dnd5eCmdRc.Solve(ctx, msg, cmdArgs)
	},
}
