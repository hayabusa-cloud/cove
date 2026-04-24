[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/cove.svg)](https://pkg.go.dev/code.hybscloud.com/cove)
[![Go Report Card](https://goreportcard.com/badge/github.com/hayabusa-cloud/cove)](https://goreportcard.com/report/github.com/hayabusa-cloud/cove)
[![Coverage Status](https://codecov.io/gh/hayabusa-cloud/cove/graph/badge.svg)](https://codecov.io/gh/hayabusa-cloud/cove)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[English](README.md) | [简体中文](README.zh-CN.md) | [Español](README.es.md) | [日本語](README.ja.md) | **Français**

# cove

Couche de contexte pour transporter un contexte ambiant explicite à travers les frontières de
suspension [kont](https://code.hybscloud.com/kont) en Go.

## Vue d’ensemble

Lorsqu’un runtime externe, tel qu’un proacteur, une boucle d’événements ou un dispatcher, traite les suspensions kont
une par une, chaque opération suspendue peut avoir besoin d’un état qui ne fait pas partie de l’opération elle-même :
budget de dispatch, capacités de l’anneau, phase du protocole, validité du groupe de buffers. Sans porteurs de contexte
explicites, cet état finit dans des tables auxiliaires ad hoc ou des globales implicites qui brisent la composition.

cove apparie le contexte ambiant aux valeurs et aux suspensions afin que le contexte traverse les frontières de stepping
asynchrone sous forme de données typées et composables.

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

Le paquet est sans politique : il transporte et vérifie le contexte, mais ne planifie, ne réessaie ni ne dispatche
jamais.

## Installation

```bash
go get code.hybscloud.com/cove
```

Nécessite Go 1.26+.

## Types principaux

| Type                                  | Rôle                                                            |
|---------------------------------------|-----------------------------------------------------------------|
| `View[C, A]`                          | Porteur comonadique : valeur A sous contexte ambiant C          |
| `SuspensionView[C, A]`                | Suspension kont appariée avec le contexte ambiant               |
| `Req[C]`                              | Prédicat contravariant sur C (forme fermeture) : `func(C) bool` |
| `ReqExpr[C]`                          | Prédicat contravariant sur C (forme défonctionnalisée)          |
| `Rule[C]` / `RuleExpr[C]`             | Prédicat nommé avec `Report` de diagnostic                      |
| `Checked[C, A]` / `CheckedExpr[C, A]` | Valeur conditionnée par une exigence                            |
| `Guarded[C, A]` / `GuardedExpr[C, A]` | Valeur conditionnée par une règle nommée                        |

`Ambient` contraint les paramètres de type de contexte (C, D). `Focus` contraint les paramètres de type de valeur (A, B).

## Stepping contextuel

`StepWith` et `StepExprWith` évaluent une computation kont et apparient sa première suspension avec le contexte ambiant.
Chaque suspension est affine : elle se consomme exactement une fois, via `Resume`, `ResumeWith` ou `Discard`.

```go
val, sv := cove.StepExprWith(ctx, expr)
for sv.Suspension != nil {
    op := sv.Op()
    result := handle(op)
    val, sv = sv.Resume(result)
}
```

`ResumeWith` fait évoluer le contexte transporté entre les pas. Par exemple, il peut décrémenter le budget, mettre à
jour des capacités ou faire avancer une phase de protocole :

```go
val, sv = sv.ResumeWith(result, func(r Runtime) Runtime {
    r.Budget--
    return r
})
```

`MapContextSuspension` et `WithContextSuspension` transportent ou remplacent explicitement le contexte porté par une
suspension déjà observée, sans modifier la frontière de suspension courante :

```go
sv = cove.MapContextSuspension(sv, func(r Runtime) Runtime {
    r.Budget += 4
    return r
})
sv = cove.WithContextSuspension(sv, Runtime{Budget: 16})
```

Une fois la computation terminée, `sv.Suspension` devient nil, mais `sv.Ask()` continue de renvoyer le contexte porté.

cove relaie le classificateur de stepping de kont, donc `Step`, `StepExpr`, `StepWith`, `StepExprWith`, `Resume` et
`ResumeWith` héritent de la convention de fin par nil de kont : une valeur finale nil dénote la fin avec la valeur
zéro de `A`. Les computations dont le type de résultat est un pointeur ou une interface ne peuvent donc pas utiliser
nil comme valeur finale significative ; encapsulez nil dans un type somme ou témoin explicite lorsque cette
distinction compte. Le contexte du porteur n’est pas affecté.

`ObserveSuspension` attache le contexte ambiant à une `kont.Suspension` existante sans effectuer aucune vérification
d’exigence. `CheckSuspension` et `CheckSuspensionExpr` en fournissent les variantes conditionnées :

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

## Commandes

`Cmd[C, A, B]` est une commande contextuelle `View[C, A] -> B`. `Run` applique une commande à une `View` concrète ;
`ExtractCmd` est la commande identité ; `LiftCmd` relève dans le monde contextuel une transformation ne portant que sur
la valeur focalisée ; `Compose` compose des commandes via `Extend`.

```go
cmd := cove.Compose(
    func(v cove.View[Runtime, int]) string {
        return fmt.Sprintf("budget=%d value=%d", v.Ask().Budget, v.Extract())
    },
    cove.LiftCmd(func(n int) int { return n + 1 }),
)
out := cove.Run(cove.Observe(Runtime{Budget: 8}, 41), cmd)
_ = out // "budget=8 value=42"
```

## Exigences

Les exigences sont des prédicats sur le contexte ambiant, disponibles aussi bien sous forme fermeture que sous forme
données. Les deux formes offrent `All`, `Any` et `Not`, avec `True` et `False` comme éléments neutres de la conjonction
et de la disjonction.

**Forme fermeture** (`Req`) : directe et concise.

```go
req := cove.All(
    cove.Req[Runtime](func(r Runtime) bool { return r.Budget > 0 }),
    cove.Req[Runtime](func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.Need(runtime, req)
```

**Forme données** (`ReqExpr`) : structure booléenne composable qui évite l’allocation de fermetures durant la
composition.

```go
expr := cove.ExprAll(
    cove.ExprAtom(func(r Runtime) bool { return r.Budget > 0 }),
    cove.ExprAtom(func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.NeedExpr(runtime, expr)
```

Combinateurs : `All`/`ExprAll` (conjonction ∧), `Any`/`ExprAny` (disjonction ∨), `Not`/`ExprNot` (négation ¬), `True`/`ExprTrue` (⊤), `False`/`ExprFalse` (⊥).

`Pullback` et `ExprPullback` transportent les exigences de façon contravariante le long d’une projection de contexte.
Étant donné `f: C → D`, `Pullback(req, f)` envoie `Req[D]` vers `Req[C]`. `ExprPullback` préserve la structure booléenne
et se distribue sur `ExprNot`, `ExprAll` et `ExprAny`.

### Choisir entre Req et ReqExpr

Utilisez `Req` pour les prédicats ponctuels et ad hoc. Utilisez `ReqExpr` lorsque vous composez de nombreuses exigences,
ou lorsque l’allocation de fermetures à la construction compte.

Les helpers de pont maintiennent les deux formes alignées : `ReifyReq` enveloppe une exigence en forme fermeture comme
un `ExprAtom`, donc `ReifyReq(nil)` est invalide et déclenche un panic à l’évaluation ; `ReflectReq` évalue une exigence
en forme Expr via une fermeture.

## Valeurs conditionnées

`Guard` apparie une valeur avec une exigence ; `IntoView` ne renvoie un `View` que lorsque le contexte ambiant satisfait
cette exigence :

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

`CheckRules` et `CheckRulesExpr` évaluent plusieurs règles dans l’ordre, en s’arrêtant au premier échec :

```go
report := cove.CheckRules(runtime, budgetRule, permRule)
if !report.OK() {
    // report.Failed nomme la première règle qui n'a pas tenu
}
```

Le diagnostic en forme Expr a la même forme à travers `CheckRulesExpr` et `GuardRuleExpr` :

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

Équivalents en forme Expr : `GuardExpr`, `GuardRuleExpr`, `CheckedExpr`, `GuardedExpr`. Helpers map et pullback :
`MapChecked`, `MapGuarded`, `PullbackChecked`, `PullbackGuarded` (et leurs variantes Expr).

## Opérations View

`View[C, A]` est une comonade sur la catégorie des types Go. Le porteur est le produit C × A ; `Extract` (ε) projette la valeur ; `Duplicate` (δ) plonge l’observation comme valeur focalisée :

```go
v := cove.Observe(ctx, value)
v.Ask()       // contexte ambiant (π₁)
v.Extract()   // valeur focalisée (ε = π₂)
```

Transformations : `Map` (relèvement fonctoriel sur `A`), `MapContext` (relèvement fonctoriel sur `C`), `Replace` (
substituer la valeur), `WithContext` (substituer le contexte).

`Duplicate` et `Extend` satisfont les lois comonadiques :

```go
// Loi de counité : ε ∘ δ = id
cove.Duplicate(v).Extract() == v

// Identité d'extension : extend(ε) = id
cove.Extend(v, func(w cove.View[C, A]) A { return w.Extract() }) == v

// Associativité : extend(f) ∘ extend(g) = extend(f ∘ extend(g))
```

`Extend` est l’opérateur d’extension contextuelle : étant donné `f: W(C, A) → B`, il relève `f` en `W(C, A) → W(C, B)`
en préservant le contexte ambiant. La composition de commandes induite avec `g: W(C, B) → D` est `g ∘ Extend(f)`.

## Position dans l’écosystème

| Paquet | Responsabilité | Aspect catégorique |
|--------|----------------|--------------------|
| `kont` | Opérations d’effets, gestionnaires, suspension et reprise | Algèbre / Monade libre / Pli |
| **`cove`** | **Contexte ambiant à travers les frontières de suspension** | **Coalgèbre / Comonade / Dépli** |
| `takt` | Dispatch proacteur, classification des résultats, boucle d’événements | Comodèle (dual du gestionnaire) |
| `uring` | I/O noyau : SQE/CQE, gestion des anneaux, anneaux de buffers | — |
| `iox` | Algèbre de résultats et sémantique des erreurs | — |

## Structure formelle

`View[C, A]` est une comonade d’environnement (Uustalu & Vene 2008) de porteur C × A, counité ε = π₂ (Extract) et
comultiplication δ(c, a) = (c, (c, a)) (Duplicate). `Extend` implémente l’extension contextuelle (extension coKleisli en
littérature catégorique).

`Req[C]` et `ReqExpr[C]` sont des objets de l’algèbre booléenne des prédicats sur C. `Pullback` implémente le foncteur contravariant f* induit par un morphisme f: C → D, envoyant les prédicats sur D vers les prédicats sur C. `ExprPullback` préserve la structure d’algèbre booléenne : f*(p ∧ q) = f*(p) ∧ f*(q), f*(¬p) = ¬f*(p).

kont modélise les effets algébriques (monade libre, gestionnaires, pli) ; cove modélise les coeffets (comonade,
exigences, dépli). Les deux sont des duaux catégoriques à la frontière de suspension : kont décrit ce que le calcul fait
au monde, tandis que cove énonce ce dont le calcul a besoin du monde.

Lu à travers le prisme des effets modaux, la forme du porteur peut se lire comme le nom de ce qui arrive au contexte
ambiant : kont (C → D) le remplace, cove (W(C, A) → B) le traite relativement, et le passage pur le préserve. La
structure des effets voyage ainsi sur le porteur, et non sur une « couleur » de fonction, ce qui garde cove libre de
toute politique.

`ReqExpr` défonctionnalise l’algèbre booléenne en une union étiquetée, à savoir l’algèbre initiale du foncteur de
signature booléenne, de sorte que la structure d’une exigence soit une donnée inspectable plutôt qu’une fermeture
opaque (Reynolds 1972).

## Schémas pratiques

Un calcul contextuel complet procède généralement en trois étapes : déclarer des exigences, conditionner des valeurs sur
celles-ci, puis observer le résultat sous un contexte concret.

```go
// 1. Déclarer une exigence sur le contexte ambiant.
type Caps struct{ CanSubmit, HasToken bool }
req := cove.All(
    func(c Caps) bool { return c.CanSubmit },
    func(c Caps) bool { return c.HasToken },
)

// 2. Conditionner une valeur sur cette exigence. Cela produit un Checked[Caps, T].
checked := cove.Guard(req, payload)

// 3. Réifier vers un Expr pour pouvoir progresser sous différents contextes ambiants.
expr := cove.ReifyReq(req)
ok   := cove.NeedExpr(Caps{CanSubmit: true, HasToken: true}, expr)
_    = checked
_    = ok
```

`Pullback` permet à un appelant d’adapter une exigence définie sur un contexte pour qu’elle s’applique sur un autre (
`Pullback(req, f)` avec `f: D → C`) ; `MapChecked` et `MapGuarded` transportent des valeurs conditionnées à travers des
transformations au niveau valeur sans re-vérifier le prédicat. La combinaison `All` + `Pullback` + `Guard` est la
manière dont les paquets en aval construisent des vérifications de capacités typées sans se coupler à un unique type de
contexte.

## Références

- John C. Reynolds. 1972. Definitional Interpreters for Higher-Order Programming Languages. In *Proc. ACM Annual
  Conference (ACM '72)*. 717–740. https://doi.org/10.1145/800194.805852
- Tarmo Uustalu and Varmo Vene. 2008. Comonadic Notions of Computation. *Electronic Notes in Theoretical Computer
  Science* 203, 5 (June 2008), 263–284. https://doi.org/10.1016/j.entcs.2008.05.029
- Tomas Petricek, Dominic Orchard, and Alan Mycroft. 2014. Coeffects: A Calculus of Context-Dependent Computation. In
  *Proc. 19th ACM SIGPLAN International Conference on Functional Programming (ICFP '14)*.
  123–135. https://tomasp.net/academic/papers/structural/coeffects-icfp.pdf
- Marco Gaboardi, Shin-ya Katsumata, Dominic Orchard, Flavien Breuvart, and Tarmo Uustalu. 2016. Combining Effects and
  Coeffects via Grading. In *Proc. 21st ACM SIGPLAN International Conference on Functional Programming (ICFP '16)*.
  476–489. https://doi.org/10.1145/2951913.2951939
- Jonathan Immanuel Brachthäuser, Philipp Schuster, and Klaus Ostermann. 2020. Effects as Capabilities: Effect Handlers
  and Lightweight Effect Polymorphism. *Proc. ACM on Programming Languages* 4, OOPSLA (Nov. 2020), Article 126, 30
  pages. https://se.cs.uni-tuebingen.de/publications/brachthaeuser20effekt.pdf
- Daniel Gratzer, G. A. Kavvos, Andreas Nuyts, and Lars Birkedal. 2021. Multimodal Dependent Type Theory. *Logical
  Methods in Computer Science* 17, 3 (2021), Paper 11, 67 pages. https://doi.org/10.46298/lmcs-17(3:11)2021
- Wenhao Tang, Leo White, Stephen Dolan, Daniel Hillerström, Sam Lindley, and Anton Lorenzen. 2025. Modal Effect Types.
  *Proc. ACM Program. Lang.* 9, OOPSLA1 (Apr. 2025), Article 120, 1130–1157. https://arxiv.org/abs/2407.11816
- Wenhao Tang and Sam Lindley. 2026. Rows and Capabilities as Modal Effects. *Proc. ACM Program. Lang.* 10, POPL (Jan.
  2026), 923–950. https://arxiv.org/abs/2507.10301

## Support de plateforme

`cove` est un paquet Go pur dans ce module. Il n'a ni fichier spécifique à une plateforme ni build tag ; la seule exigence du module est Go 1.26+.

## Licence

Licence MIT. Voir [LICENSE](LICENSE) pour les détails.

©2026 [Hayabusa Cloud Co., Ltd.](https://code.hybscloud.com)
