package dice

func RegisterBuiltinExtPmdnd(self *Dice) {
	theExt := &ExtInfo{
		Name:       "pmdnd",
		Version:    "1.1.0",
		Brief:      "提供PMDnD规则TRPG支持 - 宝可梦d20系统",
		Author:     "SealDice",
		AutoActive: true,
		Official:   true,
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
		},
		GetDescText: GetExtensionDesc,
		CmdMap: CmdMapCls{
			// 核心命令（英文）
			"pmdnd":   cmdPmdnd,
			"pmdndx":  cmdPmdnd,
			"st":      cmdSt,
			"rc":      cmdRc,
			"ring":    cmdRing,
			"rest":    cmdRest,
			"move":    cmdMove,
			"dmg":     cmdDmg,
			"type":    cmdType,
			"crit":    cmdCrit,
			"init":    cmdInit,
			"pmi":     cmdPmi,
			"npc":     cmdNPC,
			"ds":      cmdDs,
			"buff":    cmdBuff,
			"weather": cmdWeather,
			"terrain": cmdTerrain,
			"pmode":   cmdPmode,

			// 中文别名
			"属性":   cmdSt,
			"检定":   cmdRc,
			"环位":   cmdRing,
			"休息":   cmdRest,
			"招式":   cmdMove,
			"伤害":   cmdDmg,
			"克制":   cmdType,
			"暴击":   cmdCrit,
			"先攻":   cmdInit,
			"敌人":   cmdNPC,
			"濒死":   cmdDs,
			"死亡豁免": cmdDs,
			"状态":   cmdBuff,
			"天气":   cmdWeather,
			"场地":   cmdTerrain,
			"模式":   cmdPmode,
			"制卡":   cmdPmdnd,
			"长休":   cmdRestAliasLong,
			"短休":   cmdRestAliasShort,

			// 旧指令兼容
			"dst":       cmdSt,
			"ra":        cmdRc,
			"rah":       cmdRc,
			"rch":       cmdRc,
			"cast":      cmdRing,
			"longrest":  cmdRest,
			"shortrest": cmdRest,
			"ri":        cmdInit,
			"drc":       cmdRc,
			"dtype":     cmdType,
			"dcrit":     cmdCrit,
			"ddmg":      cmdDmg,
			"dmove":     cmdMove,
		},
	}
	self.RegisterExtension(theExt)
}

// 为 rest 命令创建两个别名命令，直接调用 cmdRest 但预设参数
var cmdRestAliasLong = &CmdItemInfo{
	Name: "长休",
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		cmdArgs.Args = []string{"long"}
		return cmdRest.Solve(ctx, msg, cmdArgs)
	},
}
var cmdRestAliasShort = &CmdItemInfo{
	Name: "短休",
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		cmdArgs.Args = []string{"short"}
		return cmdRest.Solve(ctx, msg, cmdArgs)
	},
}
