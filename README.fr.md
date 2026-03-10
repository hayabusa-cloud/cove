[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/cove.svg)](https://pkg.go.dev/code.hybscloud.com/cove)
[![Go Report Card](https://goreportcard.com/badge/github.com/hayabusa-cloud/cove)](https://goreportcard.com/report/github.com/hayabusa-cloud/cove)
[![Coverage Status](https://codecov.io/gh/hayabusa-cloud/cove/graph/badge.svg)](https://codecov.io/gh/hayabusa-cloud/cove)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[English](README.md) | [简体中文](README.zh-CN.md) | [Español](README.es.md) | [日本語](README.ja.md) | **Français**

# cove

Couche de contexte coalgébrique pour les suspensions [kont](https://code.hybscloud.com/kont) en Go, le dual coeffectuel des effets algébriques de kont.

## Vue d’ensemble

Lorsqu’un runtime externe — proacteur, boucle d’événements ou dispatcher — traite les suspensions kont une par une, chaque opération suspendue peut avoir besoin d’un état qui ne fait pas partie de l’opération elle-même : budget de dispatch, capacités de l’anneau, phase du protocole, validité du groupe de buffers. Sans porteurs de contexte explicites, cet état se retrouve dans des tables auxiliaires ad hoc ou des globales cachées qui brisent la composition.

cove apparie le contexte ambiant avec les valeurs et les suspensions afin que le contexte traverse les frontières de stepping asynchrone sous forme de données typées et composables.

```go
_, sv := cove.StepExprWith(Runtime{Budget: 8}, computation)
for sv.Suspension != nil {
    result := dispatch(sv.Ask(), sv.Op())
    _, sv = sv.ResumeWith(result, func(r Runtime) Runtime {
        r.Budget--
        return r
    })
}
```

Le paquet est sans politique : il transporte et vérifie le contexte mais ne planifie, ne réessaie ni ne dispatche jamais.

## Installation

```bash
go get code.hybscloud.com/cove
```

Nécessite Go 1.26+.

## Types principaux

| Type | Rôle |
|------|------|
| `View[C, A]` | Porteur comonadique : valeur A sous contexte ambiant C |
| `SuspensionView[C, A]` | Suspension kont appariée avec le contexte ambiant |
| `Req[C]` | Foncteur prédicat contravariant sur C (forme fermeture) : `func(C) bool` |
| `ReqExpr[C]` | Foncteur prédicat contravariant sur C (forme défonctionnalisée) |
| `Rule[C]` / `RuleExpr[C]` | Prédicat nommé avec `Report` de diagnostic |
| `Checked[C, A]` / `CheckedExpr[C, A]` | Valeur conditionnée par une exigence |
| `Guarded[C, A]` / `GuardedExpr[C, A]` | Valeur conditionnée par une règle nommée |

`Ambient` contraint les paramètres de type de contexte (C, D). `Focus` contraint les paramètres de type de valeur (A, B).

## Stepping contextuel

`StepWith` et `StepExprWith` évaluent une computation kont et apparient la première suspension avec le contexte ambiant. Chaque suspension est affine : elle se consomme exactement une fois via `Resume`, `ResumeWith` ou `Discard`.

```go
val, sv := cove.StepExprWith(ctx, expr)
for sv.Suspension != nil {
    op := sv.Op()
    result := handle(op)
    val, sv = sv.Resume(result)
}
```

`ResumeWith` fait évoluer le contexte entre les pas, par exemple pour décrémenter le budget, mettre à jour des capacités ou faire avancer une phase de protocole :

```go
val, sv = sv.ResumeWith(result, func(r Runtime) Runtime {
    r.Budget--
    return r
})
```

`ObserveSuspension` attache le contexte ambiant à une `kont.Suspension` existante sans vérifier d’exigence. `CheckSuspension` et `CheckSuspensionExpr` en sont les variantes conditionnées :

```go
sv, ok := cove.CheckSuspension(runtime, susp, func(r Runtime) bool {
    return r.Budget > 0
})
if !ok {
    susp.Discard()
    return
}
result := dispatch(sv.Ask(), sv.Op())
val, sv = sv.Resume(result)
```

```go
exprReq := cove.ExprAtom(func(r Runtime) bool { return r.Budget > 0 })
sv, ok := cove.CheckSuspensionExpr(runtime, susp, exprReq)
if !ok {
    susp.Discard()
    return
}
result := dispatch(sv.Ask(), sv.Op())
val, sv = sv.Resume(result)
```

`Step` et `StepExpr` réexportent les évaluateurs de kont sans lier de contexte.

## Exigences

Les exigences sont des prédicats sur le contexte, disponibles en forme fermeture et en forme données. Les deux offrent `All`/`Any`/`Not`, avec `True` et `False` comme éléments neutres de la conjonction et de la disjonction.

**Forme fermeture** (`Req`) — directe et concise :

```go
req := cove.All(
    cove.Req[Runtime](func(r Runtime) bool { return r.Budget > 0 }),
    cove.Req[Runtime](func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.Need(runtime, req)
```

**Forme données** (`ReqExpr`) — structure booléenne composable, évite l’allocation de fermetures lors de la composition :

```go
expr := cove.ExprAll(
    cove.ExprAtom(func(r Runtime) bool { return r.Budget > 0 }),
    cove.ExprAtom(func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.NeedExpr(runtime, expr)
```

Combinateurs : `All`/`ExprAll` (conjonction ∧), `Any`/`ExprAny` (disjonction ∨), `Not`/`ExprNot` (négation ¬), `True`/`ExprTrue` (⊤), `False`/`ExprFalse` (⊥).

`Pullback` et `ExprPullback` transportent les exigences de façon contravariante le long d’une projection de contexte. Étant donné f: C → D, `Pullback(req, f)` envoie `Req[D]` vers `Req[C]`. `ExprPullback` préserve la structure booléenne et se distribue sur `ExprNot`, `ExprAll` et `ExprAny`.

### Choisir entre Req et ReqExpr

Utilisez `Req` pour les prédicats ponctuels. Utilisez `ReqExpr` lorsque vous composez de nombreuses exigences ou que l’allocation de fermetures à la construction compte.

Les helpers de pont gardent les deux mondes alignés : `ReifyReq` enveloppe une exigence en forme fermeture comme `ExprAtom`, donc `ReifyReq(nil)` est invalide et déclenche un panic lors de l’évaluation ; `ReflectReq` évalue une exigence du monde Expr via une fermeture.

## Valeurs conditionnées

`Guard` apparie une valeur avec une exigence ; `IntoView` renvoie un `View` uniquement quand le contexte la satisfait :

```go
checked := cove.Guard(canDispatch, payload)
if view, ok := checked.IntoView(runtime); ok {
    _ = view.Extract()
}
```

`GuardRule` ajoute des diagnostics via des règles nommées et `Report` :

```go
rule := cove.Require("budget", func(r Runtime) bool { return r.Budget > 0 })
guarded := cove.GuardRule(rule, payload)
if view, report := guarded.IntoView(runtime); report.OK() {
    _ = view.Extract()
}
```

`CheckRules` et `CheckRulesExpr` évaluent plusieurs règles en s’arrêtant au premier échec :

```go
report := cove.CheckRules(runtime, budgetRule, permRule)
if !report.OK() {
    // report.Failed nomme la première règle qui n'a pas tenu
}
```

Le diagnostic du monde Expr suit la même forme avec `CheckRulesExpr` et `GuardRuleExpr` :

```go
type Runtime struct{ Budget int }

runtime := Runtime{Budget: 1}
payload := "packet"
exprRule := cove.RequireExpr("budget", cove.ExprAtom(func(r Runtime) bool {
    return r.Budget > 0
}))
report := cove.CheckRulesExpr(runtime, exprRule)
if !report.OK() {
    return
}
guardedExpr := cove.GuardRuleExpr(exprRule, payload)
if view, report := guardedExpr.IntoView(runtime); report.OK() {
    _ = view.Extract()
}
```

Équivalents du monde Expr : `GuardExpr`, `GuardRuleExpr`, `CheckedExpr`, `GuardedExpr`. Helpers map et pullback : `MapChecked`, `MapGuarded`, `PullbackChecked`, `PullbackGuarded` (et leurs variantes Expr).

## Opérations View

`View[C, A]` est une comonade sur la catégorie des types Go. Le porteur est le produit C × A ; `Extract` (ε) projette la valeur ; `Duplicate` (δ) plonge l’observation comme valeur focalisée :

```go
v := cove.Observe(ctx, value)
v.Ask()       // contexte ambiant (π₁)
v.Extract()   // valeur focalisée (ε = π₂)
```

Transformations : `Map` (relèvement fonctoriel sur A), `MapContext` (relèvement fonctoriel sur C), `Replace` (substituer la valeur), `WithContext` (substituer le contexte).

`Duplicate` et `Extend` satisfont les lois comonadiques :

```go
// Loi de counité : ε ∘ δ = id
cove.Duplicate(v).Extract() == v

// Identité coKleisli : extend(ε) = id
cove.Extend(v, func(w cove.View[C, A]) A { return w.Extract() }) == v

// Associativité : extend(f) ∘ extend(g) = extend(f ∘ extend(g))
```

`Extend` est l’extension coKleisli : étant donné f: W(C,A) → B, il relève f en W(C,A) → W(C,B) en préservant le contexte ambiant. La composition coKleisli induite avec g: W(C,B) → D est g ∘ Extend(f).

## Position dans l’écosystème

| Paquet | Responsabilité | Aspect catégorique |
|--------|----------------|--------------------|
| `kont` | Opérations d’effets, gestionnaires, suspension et reprise | Algèbre / Monade libre / Pli |
| **`cove`** | **Contexte ambiant à travers les frontières de suspension** | **Coalgèbre / Comonade / Dépli** |
| `takt` | Dispatch proacteur, classification des résultats, boucle d’événements | Comodèle (dual du gestionnaire) |
| `uring` | I/O noyau : SQE/CQE, gestion des anneaux, anneaux de buffers | — |
| `iox` | Algèbre de résultats et sémantique des erreurs | — |

## Structure formelle

`View[C, A]` est une comonade d’environnement (Uustalu & Vene 2008) de porteur C × A, counité ε = π₂ (Extract) et comultiplication δ(c, a) = (c, (c, a)) (Duplicate). `Extend` implémente l’extension coKleisli.

`Req[C]` et `ReqExpr[C]` sont des objets de l’algèbre booléenne des prédicats sur C. `Pullback` implémente le foncteur contravariant f* induit par un morphisme f: C → D, envoyant les prédicats sur D vers les prédicats sur C. `ExprPullback` préserve la structure d’algèbre booléenne : f*(p ∧ q) = f*(p) ∧ f*(q), f*(¬p) = ¬f*(p).

kont modélise les effets algébriques (monade libre, gestionnaires, pli). cove modélise les coeffets (comonade, exigences, dépli). Les deux sont des duaux catégoriques à la frontière de suspension : kont demande ce que le calcul fait au monde ; cove déclare ce que le calcul a besoin du monde.

`ReqExpr` défonctionnalise l’algèbre booléenne en une union étiquetée — l’algèbre initiale du foncteur de signature booléenne — de sorte que la structure des exigences soit des données inspectables plutôt que des fermetures opaques (Reynolds 1972).

## References

- Uustalu, T. & Vene, V. "Comonadic Notions of Computation." *CMCS 2008*, pp. 263–284. https://doi.org/10.1016/j.entcs.2008.05.029
- Petricek, T., Orchard, D. & Mycroft, A. "Coeffects: A Calculus of Context-Dependent Computation." *ICFP 2014*, pp. 123–135. https://doi.org/10.1145/2628136.2628160
- Reynolds, J.C. "Definitional Interpreters for Higher-Order Programming Languages." *ACM '72*, pp. 717–740. https://doi.org/10.1145/800194.805852

## Support de plateforme

`cove` est un paquet Go pur dans ce module. Il n'a ni fichier spécifique à une plateforme ni build tag ; la seule exigence du module est Go 1.26+.

## Licence

Licence MIT. Voir [LICENSE](LICENSE) pour les détails.

©2026 [Hayabusa Cloud Co., Ltd.](https://code.hybscloud.com)
