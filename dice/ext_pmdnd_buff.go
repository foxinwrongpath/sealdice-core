package dice

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	ds "github.com/sealdice/dicescript"
)

// ---------- 战斗状态结构 ----------
type BattleState struct {
	// 能力变化等级 (-6 到 +6)
	AttackLevel    int `json:"attackLevel"`
	DefenseLevel   int `json:"defenseLevel"`
	SpAttackLevel  int `json:"spAttackLevel"`
	SpDefenseLevel int `json:"spDefenseLevel"`
	SpeedLevel     int `json:"speedLevel"`

	// 护盾/结界（剩余回合数）
	ReflectWall int `json:"reflectWall"`
	LightScreen int `json:"lightScreen"`

	// 临时状态（剩余回合数）
	Protect    int `json:"protect"`
	Substitute int `json:"substitute"`

	// 天气/场地（剩余回合数）
	Weather string `json:"weather"`
	Terrain string `json:"terrain"`
}

// 能力等级对应的倍率（使用 map 支持负索引）
var abilityLevelModifiers = map[int]float64{
	-6: 0.4,
	-5: 0.5,
	-4: 0.6,
	-3: 0.7,
	-2: 0.8,
	-1: 0.9,
	0:  1.0,
	1:  1.1,
	2:  1.2,
	3:  1.3,
	4:  1.4,
	5:  1.5,
	6:  1.6,
}

// 状态名称映射
var buffNameMap = map[string]string{
	"物攻":  "attackLevel",
	"物防":  "defenseLevel",
	"特攻":  "spAttackLevel",
	"特防":  "spDefenseLevel",
	"速度":  "speedLevel",
	"反射壁": "reflectWall",
	"光墙":  "lightScreen",
	"保护":  "protect",
	"替身":  "substitute",
}

// ---------- 辅助函数 ----------
func getAbilityModifier(level int) float64 {
	if level < -6 {
		level = -6
	}
	if level > 6 {
		level = 6
	}
	return abilityLevelModifiers[level]
}

// 状态转字符串（用于显示）
func stateToString(state *BattleState) string {
	var parts []string
	if state.AttackLevel != 0 {
		parts = append(parts, fmt.Sprintf("物攻: %+d", state.AttackLevel))
	}
	if state.DefenseLevel != 0 {
		parts = append(parts, fmt.Sprintf("物防: %+d", state.DefenseLevel))
	}
	if state.SpAttackLevel != 0 {
		parts = append(parts, fmt.Sprintf("特攻: %+d", state.SpAttackLevel))
	}
	if state.SpDefenseLevel != 0 {
		parts = append(parts, fmt.Sprintf("特防: %+d", state.SpDefenseLevel))
	}
	if state.SpeedLevel != 0 {
		parts = append(parts, fmt.Sprintf("速度: %+d", state.SpeedLevel))
	}
	if state.ReflectWall > 0 {
		parts = append(parts, fmt.Sprintf("反射壁: %d回合", state.ReflectWall))
	}
	if state.LightScreen > 0 {
		parts = append(parts, fmt.Sprintf("光墙: %d回合", state.LightScreen))
	}
	if state.Protect > 0 {
		parts = append(parts, fmt.Sprintf("保护: %d回合", state.Protect))
	}
	if state.Substitute > 0 {
		parts = append(parts, fmt.Sprintf("替身: %d回合", state.Substitute))
	}
	if len(parts) == 0 {
		return "无特殊状态"
	}
	return strings.Join(parts, "  ")
}

// ---------- 加载/保存战斗状态 ----------
func loadBattleState(ctx *MsgContext) *BattleState {
	attrs, _ := ctx.Dice.AttrsManager.LoadByCtx(ctx)
	val := attrs.Load("$battle_state")
	if val == nil {
		return &BattleState{}
	}
	var state BattleState
	if err := json.Unmarshal([]byte(val.ToString()), &state); err != nil {
		return &BattleState{}
	}
	return &state
}

func saveBattleState(ctx *MsgContext, state *BattleState) {
	attrs, _ := ctx.Dice.AttrsManager.LoadByCtx(ctx)
	data, _ := json.Marshal(state)
	attrs.Store("$battle_state", ds.NewStrVal(string(data)))
}

// clampLevel 限制能力等级在 -6 到 +6 之间
func clampLevel(v int) int {
	if v > 6 {
		return 6
	}
	if v < -6 {
		return -6
	}
	return v
}

// ---------- .buff 命令 ----------
func getBuffHelp() string {
	return "PMDnD 战斗状态管理:\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		".buff stat              查看所有战斗状态\n" +
		".buff stat <名称>       查看特定状态\n" +
		".buff set <名称> <值>   设置状态值\n" +
		".buff tick              所有状态减少1回合\n" +
		".buff clear             清除所有状态\n" +
		"\n" +
		"📋 状态名称:\n" +
		"  能力等级: 物攻, 物防, 特攻, 特防, 速度 (范围 -6 到 +6)\n" +
		"  护盾结界: 反射壁, 光墙 (回合数)\n" +
		"  临时状态: 保护, 替身 (回合数)\n" +
		"\n" +
		"💡 使用强化招式时会自动更新状态:\n" +
		"  .move 剑舞 @自己       物攻 +2\n" +
		"  .move 反射壁 @自己     反射壁 5 回合\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

var cmdBuff = &CmdItemInfo{
	Name:      "buff",
	ShortHelp: ".buff stat              查看状态\n.buff set <名称> <值>  设置状态\n.buff tick             回合推进\n.buff clear            清除状态",
	Help:      getBuffHelp(),
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		sub := cmdArgs.GetArgN(1)
		switch sub {
		case "stat":
			state := loadBattleState(ctx)
			name := cmdArgs.GetArgN(2)
			if name != "" {
				// 查看特定状态
				field := buffNameMap[name]
				if field == "" {
					ReplyToSender(ctx, msg, fmt.Sprintf("未知状态: %s", name))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				var val interface{}
				switch field {
				case "attackLevel":
					val = state.AttackLevel
				case "defenseLevel":
					val = state.DefenseLevel
				case "spAttackLevel":
					val = state.SpAttackLevel
				case "spDefenseLevel":
					val = state.SpDefenseLevel
				case "speedLevel":
					val = state.SpeedLevel
				case "reflectWall":
					val = state.ReflectWall
				case "lightScreen":
					val = state.LightScreen
				case "protect":
					val = state.Protect
				case "substitute":
					val = state.Substitute
				}
				ReplyToSender(ctx, msg, fmt.Sprintf("%s: %v", name, val))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			// 查看所有状态
			playerName := getPlayerNameTempFunc(ctx)
			ReplyToSender(ctx, msg, fmt.Sprintf("📊 %s 的战斗状态:\n  %s", playerName, stateToString(state)))

		case "set":
			name := cmdArgs.GetArgN(2)
			valStr := cmdArgs.GetArgN(3)
			if name == "" || valStr == "" {
				ReplyToSender(ctx, msg, "用法: .buff set <名称> <值>")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			val, err := strconv.Atoi(valStr)
			if err != nil {
				ReplyToSender(ctx, msg, "值必须是数字")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			state := loadBattleState(ctx)
			field := buffNameMap[name]
			if field == "" {
				ReplyToSender(ctx, msg, fmt.Sprintf("未知状态: %s", name))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			switch field {
			case "attackLevel":
				if val < -6 {
					val = -6
				}
				if val > 6 {
					val = 6
				}
				state.AttackLevel = val
			case "defenseLevel":
				if val < -6 {
					val = -6
				}
				if val > 6 {
					val = 6
				}
				state.DefenseLevel = val
			case "spAttackLevel":
				if val < -6 {
					val = -6
				}
				if val > 6 {
					val = 6
				}
				state.SpAttackLevel = val
			case "spDefenseLevel":
				if val < -6 {
					val = -6
				}
				if val > 6 {
					val = 6
				}
				state.SpDefenseLevel = val
			case "speedLevel":
				if val < -6 {
					val = -6
				}
				if val > 6 {
					val = 6
				}
				state.SpeedLevel = val
			case "reflectWall":
				if val < 0 {
					val = 0
				}
				state.ReflectWall = val
			case "lightScreen":
				if val < 0 {
					val = 0
				}
				state.LightScreen = val
			case "protect":
				if val < 0 {
					val = 0
				}
				state.Protect = val
			case "substitute":
				if val < 0 {
					val = 0
				}
				state.Substitute = val
			}
			saveBattleState(ctx, state)
			ReplyToSender(ctx, msg, fmt.Sprintf("%s 设置为 %d", name, val))

		case "tick":
			state := loadBattleState(ctx)
			if state.ReflectWall > 0 {
				state.ReflectWall--
			}
			if state.LightScreen > 0 {
				state.LightScreen--
			}
			if state.Protect > 0 {
				state.Protect--
			}
			if state.Substitute > 0 {
				state.Substitute--
			}
			saveBattleState(ctx, state)
			ReplyToSender(ctx, msg, "⏱️ 状态已推进1回合")

		case "clear":
			saveBattleState(ctx, &BattleState{})
			ReplyToSender(ctx, msg, "已清除所有战斗状态")

		case "help", "":
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}

		default:
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
