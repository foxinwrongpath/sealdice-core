package dice

type DamageResult struct {
	BaseDmg  int64
	FinalDmg int64
	D20      int64
	CritText string
	RollPct  int64
	StabMul  float64
	TypeMod  float64
}

func calculateDamage(ctx *MsgContext, power int64, atkType string, isSpecial bool,
	advantage string, ctLimit int64, attacker string, defender string) (DamageResult, string) {

	var result DamageResult
	var errMsg string

	attackerCtx := ctx
	defCtx := ctx

	atkVal := int64(10)
	if isSpecial {
		if v, _ := VarGetValueInt64(attackerCtx, "satk"); v != 0 {
			atkVal = v
		}
	} else {
		if v, _ := VarGetValueInt64(attackerCtx, "patk"); v != 0 {
			atkVal = v
		}
	}

	defVal := int64(10)
	if isSpecial {
		if v, _ := VarGetValueInt64(defCtx, "sdef"); v != 0 {
			defVal = v
		}
	} else {
		if v, _ := VarGetValueInt64(defCtx, "pdef"); v != 0 {
			defVal = v
		}
	}

	battleLv := int64(1)
	if v, _ := VarGetValueInt64(attackerCtx, "战斗等级"); v != 0 {
		battleLv = v
	}

	d20Expr := "d20"
	if advantage != "" {
		d20Expr = "d20" + advantage
	}
	attackerCtx.CreateVmIfNotExists()
	r := attackerCtx.Eval(d20Expr, nil)
	if r.vm.Error != nil {
		errMsg = "骰点失败: " + r.vm.Error.Error()
		return result, errMsg
	}
	d20, _ := r.ReadInt()
	result.D20 = int64(d20)

	rollPct := int64(d20) * 5
	if int64(d20) >= ctLimit {
		rollPct += 50
		result.CritText = "【暴击+50%】"
	}
	if int64(d20) == 1 {
		rollPct = 0
		result.CritText = "【大失败】"
	}
	result.RollPct = rollPct

	totalDmg := (power * battleLv * atkVal * rollPct) / (100 * defVal)
	if totalDmg < 1 {
		totalDmg = 1
	}
	if rollPct == 0 {
		totalDmg = 0
	}
	result.BaseDmg = totalDmg

	atkTypes := []string{}
	for _, t := range pmdndTypeNames {
		if v, _ := VarGetValueInt64(attackerCtx, "$type_"+t); v > 0 {
			atkTypes = append(atkTypes, t)
		}
	}
	stabMul := 1.0
	for _, t := range atkTypes {
		if t == atkType {
			if v, _ := VarGetValueInt64(attackerCtx, "$stab_"+t); v > 0 {
				stabMul = (100.0 + float64(v)) / 100.0
			} else {
				stabMul = 1.5
			}
			break
		}
	}
	result.StabMul = stabMul

	typeMod := 0.0
	if defender != "" {
		for _, t := range pmdndTypeNames {
			if v, _ := VarGetValueInt64(defCtx, "$type_"+t); v > 0 {
				if m, ok := pmdndTypeChart[atkType][t]; ok {
					typeMod += m
				}
			}
		}
	}
	result.TypeMod = typeMod

	finalDmg := totalDmg
	if typeMod != 0 || stabMul != 1.0 {
		factor := (2.0 + typeMod) / 2.0
		if factor < 0.25 {
			factor = 0.25
		}
		factor *= stabMul
		finalDmg = int64(float64(totalDmg) * factor)
	}
	result.FinalDmg = finalDmg

	return result, ""
}
