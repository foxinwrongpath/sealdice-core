package dice

import (
	"fmt"
	"strings"
)

// ----- 数据结构定义 -----

// DamageResult 伤害计算结果
type DamageResult struct {
	BaseDmg    int64   // 基础伤害（未修正）
	FinalDmg   int64   // 最终伤害（已修正）
	D20        int64   // d20 出目（原始值）
	AttackRoll int64   // 攻击掷骰结果（d20 + 攻击加值）
	CritText   string  // 暴击/大失败文本
	RollPct    float64 // 攻击掷骰百分比 (0-1)
	StabMul    float64 // STAB 倍率
	TypeMod    float64 // 属性克制修正
	EffectText string  // 效果文本（"效果拔群！"等）
	Hit        bool    // 是否命中
	Crit       bool    // 是否暴击
	Attacker   string  // 攻击者名称
	Defender   string  // 防御者名称
	Power      int64   // 招式威力
	AtkType    string  // 攻击属性
	AtkVal     int64   // 攻击值（已应用能力等级修正）
	DefVal     int64   // 防御值
	BattleLv   int64   // 战斗等级
}

// HealResult 治疗计算结果
type HealResult struct {
	BaseHeal   int64   // 基础治疗量
	FinalHeal  int64   // 最终治疗量
	D20        int64   // d20 出目
	CritText   string  // 暴击文本
	RollPct    float64 // 治疗掷骰百分比 (0-1)
	StabMul    float64 // STAB 倍率
	Crit       bool    // 是否暴击
	HealAtkVal int64   // 治疗攻击值（特防）
}

// ----- 状态修正辅助函数 -----

// applyCumulativeStateModifiers 应用防御者的累积型状态修正
func applyCumulativeStateModifiers(defState *BattleState, isSpecial bool, rawAtkVal *int64, defVal *int64, hitPenalty *int) {
	if defState == nil {
		return
	}

	// 灼伤：物攻 -25（仅物理攻击）
	if layers, ok := defState.Cumulative["灼伤"]; ok && layers > 0 && !isSpecial {
		*rawAtkVal -= int64(layers) * 25 / 20 // 每层 -1.25，取整
	}
	// 冻伤：特攻 -25（仅特殊攻击）
	if layers, ok := defState.Cumulative["冻伤"]; ok && layers > 0 && isSpecial {
		*rawAtkVal -= int64(layers) * 25 / 20
	}
	// 溶解：物防 -25
	if layers, ok := defState.Cumulative["溶解"]; ok && layers > 0 && !isSpecial {
		*defVal -= int64(layers) * 25 / 20
	}
	// 破防：特防 -25
	if layers, ok := defState.Cumulative["破防"]; ok && layers > 0 && isSpecial {
		*defVal -= int64(layers) * 25 / 20
	}
	// 麻痹：回避 -2（影响最终伤害减免）
	if layers, ok := defState.Cumulative["麻痹"]; ok && layers > 0 {
		// 麻痹每层 -0.1 回避，最高 -2
		*hitPenalty -= int(layers / 10)
		if *hitPenalty < -2 {
			*hitPenalty = -2
		}
	}
	// 瞌睡：命中 -2（影响攻击掷骰）
	// 注意：瞌睡影响攻击方，但我们这里只处理防御者状态
	// 攻击者的瞌睡应该在调用 calculateDamage 之前由调用者处理
}

// applySevereStateEffects 应用防御者的严重状态效果
func applySevereStateEffects(defState *BattleState, isSpecial bool, rawAtkVal *int64, defVal *int64) {
	if defState == nil {
		return
	}

	// 检查是否有任何严重状态
	severeStates := map[string]string{
		"严重灼伤": "物攻",
		"严重冻伤": "特攻",
		"严重溶解": "物防",
		"严重破防": "特防",
	}

	for severeName, target := range severeStates {
		// 检查持续型状态中是否有严重状态（严重状态作为持续型状态存储）
		for _, os := range defState.Ongoing {
			if os.Name == severeName && os.Rounds > 0 {
				switch target {
				case "物攻":
					if !isSpecial {
						*rawAtkVal -= 50
					}
				case "特攻":
					if isSpecial {
						*rawAtkVal -= 50
					}
				case "物防":
					if !isSpecial {
						*defVal -= 50
					}
				case "特防":
					if isSpecial {
						*defVal -= 50
					}
				}
			}
		}
	}
}

// ----- 辅助获取函数 -----

// getAttackValue 获取攻击者的原始攻击值（未应用能力等级修正）
func getAttackValue(ctx *MsgContext, attacker string, isSpecial bool) int64 {
	attackerCtx := ctx
	val := int64(10)
	if !isSpecial {
		val = getNPCAttr(ctx, attacker, "patk")
		if val == 0 {
			if v, _ := VarGetValueInt64(attackerCtx, "patk"); v != 0 {
				val = v
			} else {
				val = 10
			}
		}
	} else {
		val = getNPCAttr(ctx, attacker, "satk")
		if val == 0 {
			if v, _ := VarGetValueInt64(attackerCtx, "satk"); v != 0 {
				val = v
			} else {
				val = 10
			}
		}
	}
	return val
}

// getDefenseValue 获取防御者的防御值
func getDefenseValue(ctx *MsgContext, defender string, isSpecial bool) int64 {
	defCtx := ctx
	val := int64(10)
	if !isSpecial {
		val = getNPCAttr(ctx, defender, "pdef")
		if val == 0 {
			if v, _ := VarGetValueInt64(defCtx, "pdef"); v != 0 {
				val = v
			} else {
				val = 10
			}
		}
	} else {
		val = getNPCAttr(ctx, defender, "sdef")
		if val == 0 {
			if v, _ := VarGetValueInt64(defCtx, "sdef"); v != 0 {
				val = v
			} else {
				val = 10
			}
		}
	}
	return val
}

// getBattleLevel 获取攻击者的战斗等级
func getBattleLevel(ctx *MsgContext, attacker string) int64 {
	attackerCtx := ctx
	// 默认30（PMDnD初始CR）
	level := int64(30)
	if v := getNPCAttr(ctx, attacker, "cr"); v > 0 {
		level = v
	} else if v, _ := VarGetValueInt64(attackerCtx, "战斗等级"); v != 0 {
		level = v
	}
	return level
}

// applyAbilityModifier 应用能力等级修正到攻击值
func applyAbilityModifier(state *BattleState, isSpecial bool, rawAtkVal int64) int64 {
	modifier := 1.0
	if isSpecial {
		modifier = getAbilityModifier(state.SpAttackLevel)
	} else {
		modifier = getAbilityModifier(state.AttackLevel)
	}
	atkVal := int64(float64(rawAtkVal) * modifier)
	if atkVal < 1 {
		atkVal = 1
	}
	return atkVal
}

// applyWeatherModifier 应用天气修正到攻击值
func applyWeatherModifier(atkType string, state *BattleState) float64 {
	weatherMod := 1.0
	switch state.Weather {
	case "大晴天", "sunny":
		if atkType == "火" {
			weatherMod = 1.5
		} else if atkType == "水" {
			weatherMod = 0.5
		}
	case "下雨", "rain":
		if atkType == "水" {
			weatherMod = 1.5
		} else if atkType == "火" {
			weatherMod = 0.5
		}
	case "沙暴", "sand":
		if atkType == "岩石" {
			weatherMod = 1.5
		}
	case "冰雹", "hail":
		if atkType == "冰" {
			weatherMod = 1.5
		}
	case "雪景", "snow":
		if atkType == "冰" {
			weatherMod = 1.5
		}
	}
	return weatherMod
}

// applyTerrainModifier 应用场地修正到攻击值
func applyTerrainModifier(atkType string, state *BattleState) float64 {
	terrainMod := 1.0
	switch state.Terrain {
	case "电气场地", "electric":
		if atkType == "电" {
			terrainMod = 1.3
		}
	case "青草场地", "grassy":
		if atkType == "草" {
			terrainMod = 1.3
		}
	case "精神场地", "psychic":
		if atkType == "超能力" {
			terrainMod = 1.3
		}
	case "薄雾场地", "misty":
		if atkType == "妖精" {
			terrainMod = 1.3
		}
	case "龙之场地", "dragon":
		if atkType == "龙" {
			terrainMod = 1.3
		}
	case "失序场地", "chaos":
		if atkType != "一般" && atkType != "力场" {
			terrainMod = 1.2
		}
	}
	return terrainMod
}

// rollD20 掷骰并返回结果和百分比
// 参数:
//   - ctx: 上下文
//   - advantage: "优势" 或 "劣势" 或 ""
//   - ctLimit: 暴击阈值
//   - attackBonus: 攻击掷骰加值（来自力量/敏捷调整值，或用户输入的 +N）
//   - hitPenalty: 命中惩罚（来自瞌睡等状态，为负数）
//
// 返回:
//   - d20: 原始骰值
//   - attackRoll: 攻击掷骰结果（d20 + attackBonus + hitPenalty）
//   - rollPct: 攻击掷骰百分比 (0-1)
//   - critText: 暴击/大失败文本
func rollD20(ctx *MsgContext, advantage string, ctLimit int64, attackBonus int64, hitPenalty int) (d20 int64, attackRoll int64, rollPct float64, critText string) {
	d20Expr := "d20"
	if advantage != "" {
		d20Expr = "d20" + advantage
	}
	ctx.CreateVmIfNotExists()
	r := ctx.Eval(d20Expr, nil)
	if r.vm.Error != nil {
		return 0, 0, 0, "骰点失败: " + r.vm.Error.Error()
	}
	d, _ := r.ReadInt()
	d20 = int64(d)

	// 计算攻击掷骰结果
	attackRoll = d20 + attackBonus + int64(hitPenalty)
	if attackRoll < 0 {
		attackRoll = 0
	}
	if attackRoll > 100 {
		attackRoll = 100
	}

	// 计算百分比
	rollPct = float64(attackRoll) * 0.05
	if rollPct > 1.0 {
		rollPct = 1.0
	}

	// 暴击判定：基于原始 d20
	if d20 >= ctLimit {
		rollPct += 0.5
		critText = "【暴击+50%】"
	}
	if d20 == 1 {
		rollPct = 0
		critText = "【大失败】"
	}
	return
}

// ----- STAB 和克制计算 -----

// calculateSTAB 计算本系加成倍率
func calculateSTAB(ctx *MsgContext, attacker string, atkType string) float64 {
	isNPC := false
	data := loadNPCData(ctx)
	if _, ok := data[attacker]; ok {
		isNPC = true
	}

	atkTypes := []string{}
	for _, t := range pmdndTypeNames {
		key := "type_" + t
		var typeVal int64

		if isNPC {
			typeVal = getNPCAttr(ctx, attacker, key)
		} else {
			if v, _ := VarGetValueInt64(ctx, key); v > 0 {
				typeVal = v
			}
		}

		if typeVal > 0 {
			atkTypes = append(atkTypes, t)
		}
	}

	stabMul := 1.0
	for _, t := range atkTypes {
		if t == atkType {
			key := "stab_" + t
			var stabVal int64

			if isNPC {
				stabVal = getNPCAttr(ctx, attacker, key)
			} else {
				if v, _ := VarGetValueInt64(ctx, key); v > 0 {
					stabVal = v
				}
			}

			if stabVal > 0 {
				stabMul = (100.0 + float64(stabVal)) / 100.0
			} else {
				stabMul = 1.5
			}
			break
		}
	}
	return stabMul
}

// calculateTypeModifier 计算属性克制修正（支持多属性叠加）
func calculateTypeModifier(ctx *MsgContext, defender string, atkType string) float64 {
	if defender == "" || defender == "目标" {
		return 0.0
	}
	typeMod := 0.0

	isNPC := false
	data := loadNPCData(ctx)
	if _, ok := data[defender]; ok {
		isNPC = true
	}

	for _, t := range pmdndTypeNames {
		key := "type_" + t
		var typeVal int64

		if isNPC {
			typeVal = getNPCAttr(ctx, defender, key)
		} else {
			if v, _ := VarGetValueInt64(ctx, key); v > 0 {
				typeVal = v
			}
		}

		if typeVal > 0 {
			if m, ok := pmdndTypeChart[atkType][t]; ok {
				typeMod += m * float64(typeVal)
			}
		}
	}
	return typeMod
}

// applyBarrierReduction 应用结界减伤（反射壁/光墙）
func applyBarrierReduction(state *BattleState, isSpecial bool, damage int64) int64 {
	if !isSpecial && state.ReflectWall > 0 {
		return damage / 2
	}
	if isSpecial && state.LightScreen > 0 {
		return damage / 2
	}
	return damage
}

// determineEffectText 判定效果文本
func determineEffectText(finalDmg int64, rollPct float64, totalFactor float64, defender string) string {
	if finalDmg == 0 && rollPct > 0 {
		return "对 " + defender + " 没有效果……"
	}
	if totalFactor >= 2.0 {
		return "效果拔群！"
	}
	if totalFactor <= 0.5 && totalFactor > 0 {
		return "效果不彰……"
	}
	return ""
}

// ----- 主伤害计算函数 -----

// calculateDamage 计算伤害
// 参数:
//   - ctx: 上下文
//   - power: 招式威力
//   - atkType: 攻击属性
//   - isSpecial: 是否为特殊攻击
//   - advantage: "优势" 或 "劣势" 或 ""
//   - ctLimit: 暴击阈值
//   - attacker: 攻击者名称
//   - defender: 防御者名称
//   - attackBonus: 攻击掷骰加值（来自力量/敏捷调整值，或用户输入的 +N）
func calculateDamage(ctx *MsgContext, power int64, atkType string, isSpecial bool,
	advantage string, ctLimit int64, attacker string, defender string, attackBonus int64) (DamageResult, string) {

	var result DamageResult

	// 1. 获取基础数值
	rawAtkVal := getAttackValue(ctx, attacker, isSpecial)
	defVal := getDefenseValue(ctx, defender, isSpecial)
	battleLv := getBattleLevel(ctx, attacker)
	result.BattleLv = battleLv

	// 2. 加载防御者的战斗状态（用于状态修正）
	defState := loadBattleStateFor(ctx, defender)

	// 3. 应用防御者的累积型状态修正（灼伤/冻伤/溶解/破防/麻痹）
	hitPenalty := 0
	applyCumulativeStateModifiers(defState, isSpecial, &rawAtkVal, &defVal, &hitPenalty)

	// 4. 应用防御者的严重状态修正
	applySevereStateEffects(defState, isSpecial, &rawAtkVal, &defVal)

	// 5. 确保防御值不低于最小值
	if defVal < 1 {
		defVal = 1
	}
	result.DefVal = defVal

	// 6. 加载攻击者状态（用于天气/场地/能力等级）
	state := loadBattleState(ctx)

	// 7. 应用天气和场地修正到攻击值
	weatherMod := applyWeatherModifier(atkType, state)
	terrainMod := applyTerrainModifier(atkType, state)
	envMod := weatherMod * terrainMod
	adjustedRawAtkVal := int64(float64(rawAtkVal) * envMod)
	if adjustedRawAtkVal < 1 {
		adjustedRawAtkVal = 1
	}

	// 8. 应用能力等级修正
	atkVal := applyAbilityModifier(state, isSpecial, adjustedRawAtkVal)
	result.AtkVal = atkVal

	// 9. 检查攻击者自身的瞌睡状态（命中-2）
	attackerState := loadBattleStateFor(ctx, attacker)
	if v, ok := attackerState.Cumulative["瞌睡"]; ok && v > 0 {
		hitPenalty -= int(v / 10)
		if hitPenalty < -2 {
			hitPenalty = -2
		}
	}

	// 10. 掷骰（传入 attackBonus）
	d20, attackRoll, rollPct, critText := rollD20(ctx, advantage, ctLimit, attackBonus, hitPenalty)
	if strings.HasPrefix(critText, "骰点") {
		return result, critText
	}
	result.D20 = d20
	result.AttackRoll = attackRoll
	result.RollPct = rollPct
	result.CritText = critText
	result.Hit = rollPct > 0
	result.Crit = critText != "" && critText != "【大失败】"

	// 11. 基础伤害
	totalDmg := int64(float64(power*battleLv*atkVal) * rollPct / (100.0 * float64(defVal)))
	if totalDmg < 1 {
		totalDmg = 1
	}
	if rollPct == 0 {
		totalDmg = 0
	}
	result.BaseDmg = totalDmg

	// 12. STAB 和克制
	stabMul := calculateSTAB(ctx, attacker, atkType)
	result.StabMul = stabMul
	typeMod := calculateTypeModifier(ctx, defender, atkType)
	result.TypeMod = typeMod

	// 13. 应用 STAB 和克制
	finalDmg := totalDmg
	if typeMod != 0 || stabMul != 1.0 {
		factor := (2.0 + typeMod) / 2.0
		if factor < 0.25 {
			factor = 0.25
		}
		factor *= stabMul
		finalDmg = int64(float64(totalDmg) * factor)
	}

	// 14. 应用结界减伤
	finalDmg = applyBarrierReduction(state, isSpecial, finalDmg)
	result.FinalDmg = finalDmg

	// 15. 填充结果字段
	result.Attacker = attacker
	result.Defender = defender
	result.Power = power
	result.AtkType = atkType

	// 16. 效果文本
	totalFactor := 1.0
	if typeMod != 0 || stabMul != 1.0 {
		factor := (2.0 + typeMod) / 2.0
		if factor < 0.25 {
			factor = 0.25
		}
		totalFactor = factor * stabMul
	}
	totalFactor = totalFactor * envMod
	if (!isSpecial && state.ReflectWall > 0) || (isSpecial && state.LightScreen > 0) {
		totalFactor = totalFactor / 2
	}
	result.EffectText = determineEffectText(finalDmg, rollPct, totalFactor, defender)

	return result, ""
}

// ----- 治疗计算函数 -----

// calculateHeal 计算治疗量
func calculateHeal(ctx *MsgContext, power int64, atkType string, advantage string, ctLimit int64, attacker string, defender string) (HealResult, string) {
	var result HealResult

	// 1. 获取治疗攻击值（特防）
	healAtkVal := getNPCAttr(ctx, attacker, "sdef")
	if healAtkVal == 0 {
		if v, _ := VarGetValueInt64(ctx, "sdef"); v != 0 {
			healAtkVal = v
		} else {
			healAtkVal = 10
		}
	}
	result.HealAtkVal = healAtkVal

	// 2. 治疗公式：挑战等级固定为 100，防御固定为 200
	battleLv := int64(100)
	defVal := int64(200)

	// 3. 治疗受青草场地影响
	state := loadBattleState(ctx)
	healMod := 1.0
	if state.Terrain == "青草场地" || state.Terrain == "grassy" {
		healMod = 1.3
	}

	// 4. 治疗掷骰（治疗不受攻击加值影响，attackBonus=0）
	d20, _, rollPct, critText := rollD20(ctx, advantage, ctLimit, 0, 0)
	if strings.HasPrefix(critText, "骰点") {
		return result, critText
	}

	// 治疗无视大失败：d20=1 时正常计算（5%），而不是0
	if d20 == 1 && rollPct == 0 {
		rollPct = 0.05
		critText = ""
	}

	result.D20 = d20
	result.RollPct = rollPct
	result.CritText = critText
	result.Crit = int64(d20) >= ctLimit

	// 5. 基础治疗量
	baseHeal := int64(float64(power*battleLv*healAtkVal) * rollPct / (100.0 * float64(defVal)))
	if baseHeal < 1 {
		baseHeal = 1
	}
	if rollPct == 0 {
		baseHeal = 0
	}
	result.BaseHeal = baseHeal

	// 6. STAB 计算
	stabMul := calculateSTAB(ctx, attacker, atkType)
	result.StabMul = stabMul

	// 7. 最终治疗量
	finalHeal := baseHeal
	if stabMul != 1.0 {
		finalHeal = int64(float64(baseHeal) * stabMul)
	}
	if healMod != 1.0 {
		finalHeal = int64(float64(finalHeal) * healMod)
	}
	result.FinalHeal = finalHeal

	return result, ""
}

// ----- 死亡豁免函数 -----

// triggerDeathSave 检查玩家 HP 是否归零，并触发死亡豁免提示
func triggerDeathSave(ctx *MsgContext, playerName string) {
	if playerName != ctx.Player.Name {
		return
	}
	hp, _ := VarGetValueInt64(ctx, "hp")
	if hp > 0 {
		return
	}
	// 检查是否已经死亡
	deathSuccess, _ := VarGetValueInt64(ctx, "DSS")
	deathFailure, _ := VarGetValueInt64(ctx, "DSF")
	if deathFailure >= 3 || deathSuccess >= 3 {
		return // 已经死亡或伤势稳定
	}
	// 发出提示
	ReplyToSender(ctx, &Message{}, fmt.Sprintf(
		"💔 %s 失去了战斗能力！\n请使用 .ds 进行濒死豁免",
		getPlayerNameTempFunc(ctx)))
}
