package dice

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func getPmdndHelp() string {
	return "PMDnD 扩展命令列表:\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		"📋 属性管理\n" +
		"  .st / .属性       属性管理 (设置/查看/删除)\n" +
		"\n" +
		"🎲 技能检定\n" +
		"  .rc               技能检定 (公开)\n" +
		"  .rah / .rch       暗骰 (结果私聊)\n" +
		"\n" +
		"💀 濒死豁免\n" +
		"  .ds / .死亡豁免   濒死豁免检定 (查看/调整状态)\n" +
		"\n" +
		"💍 资源管理\n" +
		"  .ring / .环位     设置最大可用环位（影响PP消耗）\n" +
		"  .rest             长休/短休 (.rest long/short)\n" +
		"  .长休 / .短休     快捷休息 (恢复HP和PP)\n" +
		"\n" +
		"⚔️ 战斗\n" +
		"  .dmg              伤害计算\n" +
		"  .move             招式管理 (含治疗/强化自动识别)\n" +
		"  .stab             本系加成查询\n" +
		"  .type             属性克制查询\n" +
		"  .crit / .暴击     暴击判定\n" +
		"  .buff             战斗状态管理 (能力等级/护盾/结界)\n" +
		"  .weather          天气管理 (大晴天/下雨/沙暴等)\n" +
		"  .terrain          场地管理 (电气/青草/精神等)\n" +
		"\n" +
		"👾 NPC\n" +
		"  .npc              NPC属性管理 + 招式 + 攻击\n" +
		"\n" +
		"🎯 先攻与行动\n" +
		"  .init             先攻列表管理 (含 .ri 快捷添加)\n" +
		"  .action / .行动   行动资源管理\n" +
		"\n" +
		"🎴 制卡\n" +
		"  .pmdnd            生成属性 (自由分配)\n" +
		"  .pmdndx           生成属性 (预设模式)\n" +
		"\n" +
		"💡 PP消耗规则: 招式PP = 环位 × 30\n" +
		"   使用 .st ppmax:XXX 设置最大PP，.st pp:XXX 设置当前PP\n" +
		"   详细帮助: .<命令> help  例: .dmg help\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

func getPmdndShortHelp() string {
	return ".pmdnd help     显示所有命令\n" +
		".pmdnd [数量]   生成属性（自由分配）\n" +
		".pmdndx [数量]  生成属性（预设模式）"
}

var cmdPmdnd = &CmdItemInfo{
	Name:      "pmdnd",
	ShortHelp: getPmdndShortHelp(),
	Help:      getPmdndHelp(),
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		if cmdArgs.IsArgEqual(1, "help") {
			ReplyToSender(ctx, msg, getPmdndHelp())
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		isMode2 := cmdArgs.Command == "pmdndx"
		n := cmdArgs.GetArgN(1)
		val, err := strconv.ParseInt(n, 10, 64)
		if err != nil || n == "" {
			ReplyToSender(ctx, msg, getPmdndHelp())
			return CmdExecuteResult{Matched: true, Solved: true}
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
