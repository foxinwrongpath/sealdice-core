package dice

import (
	"fmt"
	"strconv"
	"strings"

	ds "github.com/sealdice/dicescript"
)

// ---------- 状态常量 ----------

var CumulativeStateNames = []string{
	"灼伤", "冻伤", "中毒", "溶解", "破防",
	"麻痹", "瞌睡", "困惑", "畏缩", "恍惚",
	"迷离", "腐蚀", "信息素", "看破", "共振",
	"诅咒", "祝福", "流血",
	"潮湿", "干燥", "寒冷", "炎热",
}

var SevereStateMap = map[string]string{
	"灼伤":  "严重灼伤",
	"冻伤":  "严重冻伤",
	"中毒":  "剧毒",
	"溶解":  "严重溶解",
	"破防":  "严重破防",
	"麻痹":  "完全麻痹",
	"瞌睡":  "睡眠",
	"困惑":  "混乱",
	"畏缩":  "震慑",
	"恍惚":  "恐慌",
	"迷离":  "魅惑",
	"腐蚀":  "腐化",
	"信息素": "锁定",
	"看破":  "超会心",
	"共振":  "震荡",
	"寒冷":  "冻结",
	"炎热":  "融化",
}

type StateEffect struct {
	DamagePerLayer int
	StatMod        string
	Desc           string
}

var StateEffects = map[string]StateEffect{
	"灼伤": {DamagePerLayer: 25, StatMod: "物攻", Desc: "物攻变化等级-25，回合结束受25威力火属性伤害/层"},
	"冻伤": {DamagePerLayer: 25, StatMod: "特攻", Desc: "特攻变化等级-25，回合结束受25威力冰属性伤害/层"},
	"中毒": {DamagePerLayer: 50, StatMod: "", Desc: "回合结束受50威力毒属性伤害/层"},
	"溶解": {DamagePerLayer: 25, StatMod: "物防", Desc: "物防变化等级-25，回合结束受25威力酸属性伤害/层"},
	"破防": {DamagePerLayer: 25, StatMod: "特防", Desc: "特防变化等级-25，回合结束受25威力超能力属性伤害/层"},
	"麻痹": {DamagePerLayer: 0, StatMod: "回避", Desc: "回避-2，20层转为完全麻痹"},
	"瞌睡": {DamagePerLayer: 0, StatMod: "命中", Desc: "命中-2，20层转为睡眠"},
	"诅咒": {DamagePerLayer: 25, StatMod: "", Desc: "光耀/黯蚀+0.5易伤，回合结束受25威力黯蚀伤害/层"},
}

// StateChangeLevels 累积型状态对应的变化等级（固定值，不分层数）
// 严重状态覆盖普通状态的效果
var StateChangeLevels = map[string]int{
	"灼伤":  -25,
	"冻伤":  -25,
	"溶解":  -25,
	"破防":  -25,
	"严重灼伤": -50,
	"严重冻伤": -50,
	"严重溶解": -50,
	"严重破防": -50,
}

// VulnStateTypeMods 易伤类状态对应的属性伤害修正（每层）
// 正值=易伤(受伤更多), 负值=抗性(受伤更少)
var VulnStateTypeMods = map[string]map[string]float64{
	"潮湿":  {"火": -0.5, "冰": 0.5, "电": 0.5},
	"干燥":  {"水": -0.5, "火": 0.5, "地面": 0.5, "岩石": 0.5},
	"寒冷":  {"火": -0.5, "水": 0.5, "冰": 0.5, "飞行": 0.5},
	"炎热":  {"冰": -0.5, "火": 0.5},
	"困惑":  {"超能力": 0.5},
	"畏缩":  {"龙": 0.5},
	"恍惚":  {"恶": 0.5, "幽灵": 0.5},
	"迷离":  {"妖精": 0.5},
	"腐蚀":  {"毒": 0.5, "酸": 0.5},
	"信息素": {"虫": 0.5},
	"看破":  {"一般": 0.5, "格斗": 0.5},
	"共振":  {"钢": 0.5, "力场": 0.5},
	"诅咒":  {"光耀": 0.5, "黯蚀": 0.5},
	"祝福":  {"光耀": -0.5, "黯蚀": -0.5},
}

// ---------- 状态数据结构 ----------

type OngoingState struct {
	Name   string `json:"name"`
	Rounds int    `json:"rounds"`
	Source string `json:"source"`
}

type BattleState struct {
	AttackLevel    int `json:"attackLevel"`
	DefenseLevel   int `json:"defenseLevel"`
	SpAttackLevel  int `json:"spAttackLevel"`
	SpDefenseLevel int `json:"spDefenseLevel"`
	SpeedLevel     int `json:"speedLevel"`

	ReflectWall int `json:"reflectWall"`
	LightScreen int `json:"lightScreen"`
	Protect     int `json:"protect"`
	Substitute  int `json:"substitute"`

	Weather string `json:"weather"`
	Terrain string `json:"terrain"`

	Cumulative map[string]int `json:"cumulative"`
	Ongoing    []OngoingState `json:"ongoing"`
}

func NewBattleState() *BattleState {
	return &BattleState{
		Cumulative: make(map[string]int),
		Ongoing:    []OngoingState{},
	}
}

// ---------- 状态池管理 ----------

func loadAllBattleStates(ctx *MsgContext) map[string]*BattleState {
	attrs, _ := ctx.Dice.AttrsManager.LoadById(ctx.Group.GroupID)
	val := attrs.Load("$battle_states")
	if val == nil || val.TypeId != ds.VMTypeDict {
		return make(map[string]*BattleState)
	}
	dd := val.MustReadDictData()
	result := make(map[string]*BattleState)
	dd.Dict.Range(func(key string, value *ds.VMValue) bool {
		if value.TypeId == ds.VMTypeDict {
			stateDD := value.MustReadDictData()
			state := &BattleState{
				Cumulative: make(map[string]int),
				Ongoing:    []OngoingState{},
			}

			// 解析基础字段
			if v, ok := stateDD.Dict.Load("attackLevel"); ok {
				if iv, _ := v.ReadInt(); ok {
					state.AttackLevel = int(iv)
				}
			}
			if v, ok := stateDD.Dict.Load("defenseLevel"); ok {
				if iv, _ := v.ReadInt(); ok {
					state.DefenseLevel = int(iv)
				}
			}
			if v, ok := stateDD.Dict.Load("spAttackLevel"); ok {
				if iv, _ := v.ReadInt(); ok {
					state.SpAttackLevel = int(iv)
				}
			}
			if v, ok := stateDD.Dict.Load("spDefenseLevel"); ok {
				if iv, _ := v.ReadInt(); ok {
					state.SpDefenseLevel = int(iv)
				}
			}
			if v, ok := stateDD.Dict.Load("speedLevel"); ok {
				if iv, _ := v.ReadInt(); ok {
					state.SpeedLevel = int(iv)
				}
			}
			if v, ok := stateDD.Dict.Load("reflectWall"); ok {
				if iv, _ := v.ReadInt(); ok {
					state.ReflectWall = int(iv)
				}
			}
			if v, ok := stateDD.Dict.Load("lightScreen"); ok {
				if iv, _ := v.ReadInt(); ok {
					state.LightScreen = int(iv)
				}
			}
			if v, ok := stateDD.Dict.Load("protect"); ok {
				if iv, _ := v.ReadInt(); ok {
					state.Protect = int(iv)
				}
			}
			if v, ok := stateDD.Dict.Load("substitute"); ok {
				if iv, _ := v.ReadInt(); ok {
					state.Substitute = int(iv)
				}
			}
			if v, ok := stateDD.Dict.Load("weather"); ok {
				state.Weather = v.ToString()
			}
			if v, ok := stateDD.Dict.Load("terrain"); ok {
				state.Terrain = v.ToString()
			}

			// 解析累积型状态
			if v, ok := stateDD.Dict.Load("cumulative"); ok && v.TypeId == ds.VMTypeDict {
				cdd := v.MustReadDictData()
				cdd.Dict.Range(func(k string, vv *ds.VMValue) bool {
					if iv, ok := vv.ReadInt(); ok {
						state.Cumulative[k] = int(iv)
					}
					return true
				})
			}

			// 解析持续型状态
			if v, ok := stateDD.Dict.Load("ongoing"); ok && v.TypeId == ds.VMTypeArray {
				arr := v.MustReadArray()
				for _, item := range arr.List {
					if item.TypeId == ds.VMTypeDict {
						idd := item.MustReadDictData()
						os := OngoingState{}
						if vv, ok := idd.Dict.Load("name"); ok {
							os.Name = vv.ToString()
						}
						if vv, ok := idd.Dict.Load("rounds"); ok {
							if iv, _ := vv.ReadInt(); ok {
								os.Rounds = int(iv)
							}
						}
						if vv, ok := idd.Dict.Load("source"); ok {
							os.Source = vv.ToString()
						}
						state.Ongoing = append(state.Ongoing, os)
					}
				}
			}

			result[key] = state
		}
		return true
	})
	return result
}

func saveAllBattleStates(ctx *MsgContext, states map[string]*BattleState) {
	attrs, _ := ctx.Dice.AttrsManager.LoadById(ctx.Group.GroupID)
	dict := ds.NewDictValWithArrayMust() // *VMDictValue

	for name, state := range states {
		subDict := ds.NewDictValWithArrayMust() // *VMDictValue

		// 将 subDict 转换为 *VMValue 才能调用 MustReadDictData
		subDictVM := (*ds.VMValue)(subDict)
		subDictVM.MustReadDictData().Dict.Store("attackLevel", ds.NewIntVal(ds.IntType(state.AttackLevel)))
		subDictVM.MustReadDictData().Dict.Store("defenseLevel", ds.NewIntVal(ds.IntType(state.DefenseLevel)))
		subDictVM.MustReadDictData().Dict.Store("spAttackLevel", ds.NewIntVal(ds.IntType(state.SpAttackLevel)))
		subDictVM.MustReadDictData().Dict.Store("spDefenseLevel", ds.NewIntVal(ds.IntType(state.SpDefenseLevel)))
		subDictVM.MustReadDictData().Dict.Store("speedLevel", ds.NewIntVal(ds.IntType(state.SpeedLevel)))
		subDictVM.MustReadDictData().Dict.Store("reflectWall", ds.NewIntVal(ds.IntType(state.ReflectWall)))
		subDictVM.MustReadDictData().Dict.Store("lightScreen", ds.NewIntVal(ds.IntType(state.LightScreen)))
		subDictVM.MustReadDictData().Dict.Store("protect", ds.NewIntVal(ds.IntType(state.Protect)))
		subDictVM.MustReadDictData().Dict.Store("substitute", ds.NewIntVal(ds.IntType(state.Substitute)))
		subDictVM.MustReadDictData().Dict.Store("weather", ds.NewStrVal(state.Weather))
		subDictVM.MustReadDictData().Dict.Store("terrain", ds.NewStrVal(state.Terrain))

		// 累积型状态
		cumDict := ds.NewDictValWithArrayMust()
		cumDictVM := (*ds.VMValue)(cumDict)
		for k, v := range state.Cumulative {
			if v > 0 {
				cumDictVM.MustReadDictData().Dict.Store(k, ds.NewIntVal(ds.IntType(v)))
			}
		}
		subDictVM.MustReadDictData().Dict.Store("cumulative", cumDictVM)

		// 持续型状态
		ongoingArr := ds.NewArrayVal()
		for _, os := range state.Ongoing {
			if os.Rounds <= 0 {
				continue
			}
			osDict := ds.NewDictValWithArrayMust()
			osDictVM := (*ds.VMValue)(osDict)
			osDictVM.MustReadDictData().Dict.Store("name", ds.NewStrVal(os.Name))
			osDictVM.MustReadDictData().Dict.Store("rounds", ds.NewIntVal(ds.IntType(os.Rounds)))
			osDictVM.MustReadDictData().Dict.Store("source", ds.NewStrVal(os.Source))
			ongoingArr.MustReadArray().List = append(ongoingArr.MustReadArray().List, osDictVM)
		}
		subDictVM.MustReadDictData().Dict.Store("ongoing", ongoingArr)

		// 将 subDict 存入 dict（转换为 *VMValue）
		dictVM := (*ds.VMValue)(dict)
		dictVM.MustReadDictData().Dict.Store(name, subDictVM)
	}

	// 将 dict 存入 attrs
	attrs.Store("$battle_states", (*ds.VMValue)(dict))
}

// loadBattleStateFor 获取指定目标的战斗状态（不存在则创建）
func loadBattleStateFor(ctx *MsgContext, target string) *BattleState {
	states := loadAllBattleStates(ctx)
	if state, ok := states[target]; ok {
		return state
	}
	state := NewBattleState()
	states[target] = state
	saveAllBattleStates(ctx, states)
	return state
}

// saveBattleStateFor 保存指定目标的战斗状态
func saveBattleStateFor(ctx *MsgContext, target string, state *BattleState) {
	states := loadAllBattleStates(ctx)
	states[target] = state
	saveAllBattleStates(ctx, states)
}

// loadBattleState 保留旧接口，加载当前用户状态
func loadBattleState(ctx *MsgContext) *BattleState {
	return loadBattleStateFor(ctx, ctx.Player.Name)
}

// saveBattleState 保留旧接口，保存当前用户状态
func saveBattleState(ctx *MsgContext, state *BattleState) {
	saveBattleStateFor(ctx, ctx.Player.Name, state)
}

// ---------- 状态辅助函数 ----------

// executeBuffTick 执行状态推进，返回结算结果的描述文本
func executeBuffTick(ctx *MsgContext) string {
	states := loadAllBattleStates(ctx)
	var results []string

	for target, state := range states {
		// 1. 结算累积型状态效果
		effectMsg := applyCumulativeEffect(ctx, target, state)
		if effectMsg != "" {
			results = append(results, fmt.Sprintf("%s: %s", target, effectMsg))
		}

		// 2. 累积型状态衰减
		tickCumulativeStates(state)

		// 3. 持续型状态衰减
		tickOngoingStates(state)

		// 4. 护盾/结界回合衰减
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

		saveBattleStateFor(ctx, target, state)
	}

	if len(results) == 0 {
		return ""
	}
	return strings.Join(results, "\n")
}

// getAbilityModifier 根据 PMDnD 规则书变化等级公式计算倍率
// x > 0: (100 + x) / 100, x ≤ 0: 100 / (100 - x)
func getAbilityModifier(level int) float64 {
	if level > 70 {
		level = 70
	}
	if level < -70 {
		level = -70
	}
	if level > 0 {
		return float64(100+level) / 100.0
	}
	return 100.0 / float64(100-level)
}

func clampLevel(v int) int {
	if v > 70 {
		return 70
	}
	if v < -70 {
		return -70
	}
	return v
}

func isCumulativeState(name string) bool {
	for _, n := range CumulativeStateNames {
		if n == name {
			return true
		}
	}
	return false
}

func getSevereState(name string) string {
	if s, ok := SevereStateMap[name]; ok {
		return s
	}
	return ""
}

func applyCumulativeEffect(ctx *MsgContext, target string, state *BattleState) string {
	var effects []string

	targetCR := int64(30)
	targetPdef := int64(10)
	targetSdef := int64(10)

	getAttr := func(key string) int64 {
		if val := getNPCAttr(ctx, target, key); val > 0 {
			return val
		}
		if target == ctx.Player.Name {
			if v, _ := VarGetValueInt64(ctx, key); v > 0 {
				return v
			}
		}
		return 0
	}

	for _, key := range []string{"挑战等级", "cr", "战斗等级"} {
		if v := getAttr(key); v > 0 {
			targetCR = v
			break
		}
	}
	if v := getAttr("pdef"); v > 0 {
		targetPdef = v
	}
	if v := getAttr("sdef"); v > 0 {
		targetSdef = v
	}

	for name, layers := range state.Cumulative {
		if layers <= 0 {
			continue
		}
		if layers >= 20 {
			severeName := getSevereState(name)
			if severeName != "" {
				state.Cumulative[name] = 10
				effects = append(effects, fmt.Sprintf("%s 转为 %s！", name, severeName))
				continue
			}
		}
		if effect, ok := StateEffects[name]; ok {
			if effect.DamagePerLayer > 0 {
				// 选择防御值（状态属性决定物/特）
				defRaw := targetPdef
				if name == "破防" || name == "诅咒" {
					defRaw = targetSdef
				}
				if defRaw < 1 {
					defRaw = 1
				}
				// 状态伤害公式: power威力 × CR² / (100 × defRaw)
				totalPower := int64(layers * effect.DamagePerLayer)
				statusDmg := totalPower * targetCR * targetCR / (100 * defRaw)
				if statusDmg < 1 {
					statusDmg = 1
				}

				damage := statusDmg
				if target == ctx.Player.Name {
					if hpMax, exists := VarGetValueInt64(ctx, "hpmax"); exists && hpMax > 0 {
						curHp, _ := VarGetValueInt64(ctx, "hp")
						newHp := curHp - damage
						if newHp < 0 {
							newHp = 0
						}
						VarSetValueInt64(ctx, "hp", newHp)
						effects = append(effects, fmt.Sprintf("%s 受到 %d 点 %s 状态伤害", target, damage, name))
					}
				} else {
					if newHp, maxHp, ok := updateNPCHP(ctx, target, damage); ok && maxHp > 0 {
						effects = append(effects, fmt.Sprintf("%s 受到 %d 点 %s 状态伤害 (HP: %d/%d)", target, damage, name, newHp, maxHp))
					}
				}
			}
		}
	}

	return strings.Join(effects, "\n")
}

func tickCumulativeStates(state *BattleState) {
	for name, layers := range state.Cumulative {
		if layers > 0 {
			newLayers := layers - 1
			if newLayers < 0 {
				newLayers = 0
			}
			state.Cumulative[name] = newLayers
		}
	}
	for name, layers := range state.Cumulative {
		if layers <= 0 {
			delete(state.Cumulative, name)
		}
	}
}

func tickOngoingStates(state *BattleState) {
	var remaining []OngoingState
	for _, os := range state.Ongoing {
		if os.Rounds > 0 {
			os.Rounds--
			if os.Rounds > 0 {
				remaining = append(remaining, os)
			}
		}
	}
	state.Ongoing = remaining
}

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

	if state.Weather != "" {
		parts = append(parts, fmt.Sprintf("天气: %s", state.Weather))
	}
	if state.Terrain != "" {
		parts = append(parts, fmt.Sprintf("场地: %s", state.Terrain))
	}

	if len(state.Cumulative) > 0 {
		var cumParts []string
		for name, layers := range state.Cumulative {
			if layers > 0 {
				cumParts = append(cumParts, fmt.Sprintf("%s:%d", name, layers))
			}
		}
		if len(cumParts) > 0 {
			parts = append(parts, "累积: "+strings.Join(cumParts, ", "))
		}
	}

	if len(state.Ongoing) > 0 {
		var ongParts []string
		for _, os := range state.Ongoing {
			if os.Rounds > 0 {
				ongParts = append(ongParts, fmt.Sprintf("%s(%d回合)", os.Name, os.Rounds))
			}
		}
		if len(ongParts) > 0 {
			parts = append(parts, "持续: "+strings.Join(ongParts, ", "))
		}
	}

	if len(parts) == 0 {
		return "无特殊状态"
	}
	return strings.Join(parts, "  ")
}

// getBuffStateHelp 返回指定状态的效果描述
func getBuffStateHelp(name string) string {
	if effect, ok := StateEffects[name]; ok {
		dmgText := ""
		if effect.DamagePerLayer > 0 {
			dmgText = fmt.Sprintf("\n  回合开始受到 %d 威力状态伤害/层", effect.DamagePerLayer)
		}
		statText := ""
		if lv, ok := StateChangeLevels[name]; ok {
			statText = fmt.Sprintf("\n  变化等级 %d", lv)
		}
		severeText := ""
		if s, ok := SevereStateMap[name]; ok {
			severeText = fmt.Sprintf("\n  20层 → %s", s)
		}
		return fmt.Sprintf("📋 %s:\n  %s%s%s%s", name, effect.Desc, statText, dmgText, severeText)
	}
	if mods, ok := VulnStateTypeMods[name]; ok {
		var parts []string
		for t, v := range mods {
			sign := "+"
			if v < 0 {
				sign = ""
			}
			parts = append(parts, fmt.Sprintf("%s%s%.1f", t, sign, v))
		}
		severeText := ""
		if s, ok := SevereStateMap[name]; ok {
			severeText = fmt.Sprintf("\n  20层 → %s", s)
		}
		return fmt.Sprintf("📋 %s (易伤类):\n  属性修正: %s%s\n  每回合减少1层", name, strings.Join(parts, " "), severeText)
	}
	return fmt.Sprintf("❌ 未知状态: %s\n  可用: .buff help 查看全部", name)
}

// ---------- .buff 命令 ----------

var cmdBuff = &CmdItemInfo{
	Name:      "buff",
	ShortHelp: ".buff stat [目标]             查看状态\n.buff add <目标> <状态> <层数>  添加状态\n.buff tick                    推进回合\n.buff set <名称> <值>         设置能力等级",
	Help:      getBuffHelp(),
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		sub := cmdArgs.GetArgN(1)

		switch sub {
		case "stat":
			target := cmdArgs.GetArgN(2)
			if target == "" {
				target = ctx.Player.Name
			}
			state := loadBattleStateFor(ctx, target)
			ReplyToSender(ctx, msg, fmt.Sprintf("📊 %s 的状态:\n  %s", target, stateToString(state)))
			return CmdExecuteResult{Matched: true, Solved: true}

		case "add":
			target := cmdArgs.GetArgN(2)
			stateName := cmdArgs.GetArgN(3)
			layersStr := cmdArgs.GetArgN(4)

			if target == "" || stateName == "" || layersStr == "" {
				ReplyToSender(ctx, msg, "用法: .buff add <目标> <状态> <层数>")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if !isCumulativeState(stateName) {
				ReplyToSender(ctx, msg, fmt.Sprintf("未知状态: %s，可用: %s", stateName, strings.Join(CumulativeStateNames, ", ")))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			layers, err := strconv.Atoi(layersStr)
			if err != nil || layers <= 0 {
				ReplyToSender(ctx, msg, "层数必须是正整数")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			state := loadBattleStateFor(ctx, target)
			current := state.Cumulative[stateName]
			newLayers := current + layers
			if newLayers > 20 {
				newLayers = 20
			}
			state.Cumulative[stateName] = newLayers
			saveBattleStateFor(ctx, target, state)

			ReplyToSender(ctx, msg, fmt.Sprintf("✅ %s 获得 %s %d 层 (当前 %d 层)", target, stateName, layers, newLayers))
			return CmdExecuteResult{Matched: true, Solved: true}

		case "remove":
			target := cmdArgs.GetArgN(2)
			stateName := cmdArgs.GetArgN(3)
			layersStr := cmdArgs.GetArgN(4)

			if target == "" || stateName == "" || layersStr == "" {
				ReplyToSender(ctx, msg, "用法: .buff remove <目标> <状态> <层数>")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			state := loadBattleStateFor(ctx, target)
			current, ok := state.Cumulative[stateName]
			if !ok || current <= 0 {
				ReplyToSender(ctx, msg, fmt.Sprintf("%s 没有 %s 状态", target, stateName))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			layers, err := strconv.Atoi(layersStr)
			if err != nil || layers <= 0 {
				ReplyToSender(ctx, msg, "层数必须是正整数")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			newLayers := current - layers
			if newLayers < 0 {
				newLayers = 0
			}
			if newLayers == 0 {
				delete(state.Cumulative, stateName)
			} else {
				state.Cumulative[stateName] = newLayers
			}
			saveBattleStateFor(ctx, target, state)

			ReplyToSender(ctx, msg, fmt.Sprintf("✅ %s 的 %s 减少 %d 层 (剩余 %d 层)", target, stateName, layers, newLayers))
			return CmdExecuteResult{Matched: true, Solved: true}

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
			switch name {
			case "物攻", "attack":
				state.AttackLevel = clampLevel(val)
			case "物防", "defense":
				state.DefenseLevel = clampLevel(val)
			case "特攻", "spattack":
				state.SpAttackLevel = clampLevel(val)
			case "特防", "spdefense":
				state.SpDefenseLevel = clampLevel(val)
			case "速度", "speed":
				state.SpeedLevel = clampLevel(val)
			case "反射壁":
				state.ReflectWall = val
				if state.ReflectWall < 0 {
					state.ReflectWall = 0
				}
			case "光墙":
				state.LightScreen = val
				if state.LightScreen < 0 {
					state.LightScreen = 0
				}
			default:
				ReplyToSender(ctx, msg, fmt.Sprintf("未知状态: %s", name))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			saveBattleState(ctx, state)
			ReplyToSender(ctx, msg, fmt.Sprintf("%s 设置为 %d", name, val))
			return CmdExecuteResult{Matched: true, Solved: true}

		case "tick":
			results := executeBuffTick(ctx)
			if results == "" {
				ReplyToSender(ctx, msg, "⏱️ 状态已推进1回合，无特殊效果")
			} else {
				ReplyToSender(ctx, msg, "⏱️ 状态已推进1回合:\n"+results)
			}
			return CmdExecuteResult{Matched: true, Solved: true}

		case "clear":
			target := cmdArgs.GetArgN(2)
			if target == "" {
				target = ctx.Player.Name
			}
			if target == "all" {
				saveAllBattleStates(ctx, make(map[string]*BattleState))
				ReplyToSender(ctx, msg, "已清除所有目标的战斗状态")
			} else {
				saveBattleStateFor(ctx, target, NewBattleState())
				ReplyToSender(ctx, msg, fmt.Sprintf("已清除 %s 的战斗状态", target))
			}
			return CmdExecuteResult{Matched: true, Solved: true}

		case "reset":
			// 重置能力变化等级（战斗结束的自然恢复）
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
			ReplyToSender(ctx, msg, "📊 能力变化等级已重置（战斗结束）\n💡 累积型状态（灼伤、中毒等）仍然存在，请根据需要治疗")
			return CmdExecuteResult{Matched: true, Solved: true}

		case "help":
			arg2 := cmdArgs.GetArgN(2)
			if arg2 != "" {
				ReplyToSender(ctx, msg, getBuffStateHelp(arg2))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}

		default:
			target := sub
			if target != "" {
				state := loadBattleStateFor(ctx, target)
				ReplyToSender(ctx, msg, fmt.Sprintf("📊 %s 的状态:\n  %s", target, stateToString(state)))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		}
	},
}
