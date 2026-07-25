package dice

func RegisterBuiltinExtPmdnd(self *Dice) {
	theExt := &ExtInfo{
		Name:       "pmdnd",
		Version:    "1.1.0",
		Brief:      "提供PMDnD规则TRPG支持 - 宝可梦主题d20系统（标准化指令）",
		Author:     "SealDice",
		AutoActive: true,
		Official:   true,
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
		},
		GetDescText: GetExtensionDesc,
		CmdMap: CmdMapCls{
			// 核心命令（标准化）
			"st":     cmdSt,
			"rc":     cmdRc,
			"ring":   cmdRing,
			"rest":   cmdRest,
			"move":   cmdMove,
			"dmg":    cmdDmg,
			"stab":   cmdStab,
			"type":   cmdType,
			"crit":   cmdCrit,
			"init":   cmdInit,
			"action": cmdAction,
			"gen":    cmdGen,

			// 中文别名（保留方便习惯）
			"属性": cmdSt,
			"环位": cmdRing,
			"长休": cmdRestAliasLong,
			"短休": cmdRestAliasShort,
			"暴击": cmdCrit,
			"行动": cmdAction,

			// 旧指令保留以兼容（但不推荐）
			"dst":       cmdSt,
			"ra":        cmdRc,
			"rah":       cmdRc,
			"rch":       cmdRc,
			"cast":      cmdRing,
			"longrest":  cmdRest,
			"shortrest": cmdRest,
			"pmdnd":     cmdGen,
			"pmdndx":    cmdGen,
			"ri":        cmdInit,
			"drc":       cmdRc,
			"dstab":     cmdStab,
			"dtype":     cmdType,
			"dcrit":     cmdCrit,
			"ddmg":      cmdDmg,
			"dmove":     cmdMove,
			"daction":   cmdAction,
		},
	}
	self.RegisterExtension(theExt)
}

// 为 rest 命令创建两个别名命令，直接调用 cmdRest 但预设参数
var cmdRestAliasLong = &CmdItemInfo{
	Name: "长休",
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		// 构造参数为 "long"
		cmdArgs.Args = []string{"rest", "long"}
		return cmdRest.Solve(ctx, msg, cmdArgs)
	},
}
var cmdRestAliasShort = &CmdItemInfo{
	Name: "短休",
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		cmdArgs.Args = []string{"rest", "short"}
		return cmdRest.Solve(ctx, msg, cmdArgs)
	},
}
