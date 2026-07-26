package dice

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	ds "github.com/sealdice/dicescript"
)

var cmdInit = &CmdItemInfo{
	Name: "init",
	ShortHelp: ".init // 查看先攻列表\n" +
		".ri <名> [d20表达式]  // 快捷添加 (d20)\n" +
		".pmi <名>            // 快捷添加 (PMDnD公式: d10+敏捷+速度因子)\n" +
		".init del <单位1> ... // 从先攻列表中删除\n" +
		".init set <单位> <表达式> // 设置单位的先攻\n" +
		".init clear // 清除先攻列表\n" +
		".init end   // 结束一回合",
	Help: getInitHelp(),
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		// .ri / .pmi 快捷添加先攻，跳过前缀解析防止 RawArgs 空切片崩溃
		if cmdArgs.Command == "ri" || cmdArgs.Command == "pmi" {
			args := cmdArgs.Args
			if len(args) == 0 {
				args = []string{ctx.Player.Name}
			}
			return handleInitAdd(ctx, msg, cmdArgs, args, cmdArgs.Command == "pmi")
		}
		cmdArgs.ChopPrefixToArgsWith("add", "list", "del", "set", "next", "clear", "clr")
		sub := cmdArgs.GetArgN(1)

		switch sub {
		// .init 查看列表
		case "", "list":
			var textOut strings.Builder
			textOut.WriteString("⚡ 行动顺序！\n")
			riList := (RIList{}).LoadByCurGroup(ctx)
			round, _ := VarGetValueInt64(ctx, "$g回合数")

			for order, i := range riList {
				arrow := "  "
				if int64(order) == round {
					arrow = "👉"
				}
				var emoji string
				if order == 0 {
					emoji = "🚩"
				} else if order == len(riList)-1 {
					emoji = "🏁"
				} else {
					emoji = "🟢"
				}
				_, _ = fmt.Fprintf(&textOut, "%s %s %d. %s: %d\n", arrow, emoji, order+1, i.name, i.val)
			}
			if len(riList) == 0 {
				textOut.WriteString("没有找到任何单位")
			} else {
				rounder := riList[round]
				_, _ = fmt.Fprintf(&textOut, "\n👉 当前回合：%s", rounder.name)
			}
			ReplyToSender(ctx, msg, textOut.String())

		case "add":
			return handleInitAdd(ctx, msg, cmdArgs, cmdArgs.Args[1:], false)

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

		case "ed", "end", "next":
			// 检查是否带 --no-tick 参数
			noTick := false
			for _, arg := range cmdArgs.Args {
				if arg == "--no-tick" || arg == "-n" {
					noTick = true
					break
				}
			}

			tickText := ""
			if !noTick {
				tickResults := executeBuffTick(ctx)
				if tickResults != "" {
					tickText = "\n⏱️ 回合结束，状态结算:\n" + tickResults
				}
			}

			// 原有的切换回合逻辑
			lst := (RIList{}).LoadByCurGroup(ctx)
			round, _ := VarGetValueInt64(ctx, "$g回合数")
			if len(lst) == 0 {
				ReplyToSender(ctx, msg, "先攻列表为空")
				break
			}
			round = (round + 1) % int64(len(lst))
			setInitNextRoundVars(ctx, lst, round)
			ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "DND:先攻_下一回合")+tickText)

		case "clr", "clear":
			// 1. 清空先攻列表
			(RIList{}).SaveToGroup(ctx)
			VarSetValueInt64(ctx, "$g当前回合先攻值", NULL_INIT_VAL)
			VarSetValueInt64(ctx, "$g回合数", 0)

			// 2. 重置所有目标的能力变化等级（战斗结束，能力恢复）
			states := loadAllBattleStates(ctx)
			for target, state := range states {
				state.AttackLevel = 0
				state.DefenseLevel = 0
				state.SpAttackLevel = 0
				state.SpDefenseLevel = 0
				state.SpeedLevel = 0
				// 注意：不清理累积型状态！
				saveBattleStateFor(ctx, target, state)
			}

			ReplyToSender(ctx, msg, "📊 先攻列表已清除\n能力变化等级已重置\n")

		default:
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}

// handleInitAdd 处理 .init add / .ri / .pmi 的先攻添加逻辑
func handleInitAdd(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs, args []string, usePMDnDFormula bool) CmdExecuteResult {
	if len(args) == 0 {
		return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
	}

	var name string
	var val int64
	isPMDnDFormula := false

	if !usePMDnDFormula {
		if num, err := strconv.ParseInt(args[0], 10, 64); err == nil {
			val = num
			if len(args) > 1 {
				name = args[1]
			} else {
				name = ctx.Player.Name
			}
		}
	}

	if name == "" {
		name = args[0]
		if len(args) > 1 && !usePMDnDFormula {
			if num2, err := strconv.ParseInt(args[1], 10, 64); err == nil {
				val = num2
			} else {
				val = int64(ds.Roll(nil, 20, 0))
			}
		} else if usePMDnDFormula {
			dex, spd := getPMDnDInitiativeStats(ctx, name)
			speedFactor := 0.0
			if spd > 1 {
				speedFactor = math.Log(float64(spd)) / math.Log(1.2)
			}
			baseInit := int64(math.Floor(float64(dex) + speedFactor))
			d10 := int64(ds.Roll(nil, 10, 0))
			val = baseInit + d10
			isPMDnDFormula = true
		} else {
			val = int64(ds.Roll(nil, 20, 0))
		}
	}

	isPlayer := name == ctx.Player.Name
	isNPC := false
	if ctx.Group != nil {
		npcData := loadNPCData(ctx)
		_, isNPC = npcData[name]
	}

	if !isPlayer && !isNPC {
		if ctx.Group == nil {
			ReplyToSender(ctx, msg, "❌ 私聊中只能为自己添加先攻\n  用法: .ri 或 .pmi (不指定名称则默认为自己)")
		} else {
			ReplyToSender(ctx, msg, fmt.Sprintf("❌ %s 不存在，请先创建:\n  冒险者: 无需创建，直接使用\n  NPC: .npc new %s", name, name))
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	}

	riList := (RIList{}).LoadByCurGroup(ctx)
	if len(riList) == 0 {
		VarSetValueInt64(ctx, "$g当前回合先攻值", NULL_INIT_VAL)
	}
	// 检查是否已存在，存在则更新值
	found := false
	for _, item := range riList {
		if item.name == name {
			item.val = val
			found = true
			break
		}
	}
	if !found {
		riList = append(riList, &RIListItem{name: name, val: val, detail: "", uid: ""})
	}
	sort.Sort(riList)
	riList.SaveToGroup(ctx)
	if isPMDnDFormula {
		dex2, spd2 := getPMDnDInitiativeStats(ctx, name)
		if found {
			ReplyToSender(ctx, msg, fmt.Sprintf("🔄 更新先攻(PMDnD): %s = %d (敏捷%d + log₁.₂(spd%d) → d10)", name, val, dex2, spd2))
		} else {
			ReplyToSender(ctx, msg, fmt.Sprintf("✅ 添加先攻(PMDnD): %s = %d (敏捷%d + log₁.₂(spd%d) → d10)", name, val, dex2, spd2))
		}
	} else {
		if found {
			ReplyToSender(ctx, msg, fmt.Sprintf("🔄 更新先攻: %s = %d", name, val))
		} else {
			ReplyToSender(ctx, msg, fmt.Sprintf("✅ 添加先攻: %s = %d", name, val))
		}
	}
	return CmdExecuteResult{Matched: true, Solved: true}
}

// .ri 快捷指令
var cmdRi = &CmdItemInfo{
	Name: "ri",
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		// 直接转发给 cmdInit，通过命令名触发 add 逻辑
		cmdArgs.Command = "ri"
		return cmdInit.Solve(ctx, msg, cmdArgs)
	},
}

// .pmi 快捷指令 — PMDnD 先攻公式
var cmdPmi = &CmdItemInfo{
	Name: "pmi",
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		cmdArgs.Command = "pmi"
		return cmdInit.Solve(ctx, msg, cmdArgs)
	},
}

// getPMDnDInitiativeStats 读取角色/NPC 的敏捷和速度属性值
func getPMDnDInitiativeStats(ctx *MsgContext, name string) (dex int64, spd int64) {
	dex = getAttrValue(ctx, name, "敏捷", "dex", "Dexterity")
	if dex < 1 {
		dex = 10
	}
	spd = getAttrValue(ctx, name, "spd", "速度", "speed")
	if spd < 1 {
		spd = 10
	}
	return
}
