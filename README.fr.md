[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/cove.svg)](https://pkg.go.dev/code.hybscloud.com/cove)
[![Benchmarks](https://github.com/hayabusa-cloud/cove/actions/workflows/benchmarks.yml/badge.svg)](https://github.com/hayabusa-cloud/cove/actions/workflows/benchmarks.yml)
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
for sv.Suspension != nil && sv.Ask().Budget > 0 {
    result := dispatch(sv.Ask(), sv.Op())
    _, sv = sv.ResumeWith(result, func(r Runtime) Runtime {
        r.Budget--
        return r
    })
}
```

Si `sv.Suspension != nil` à la sortie de la boucle, conservez `sv` et renouvelez son budget avant le prochain appel à `dispatch`.

Le paquet est sans politique : il transporte et vérifie le contexte, mais ne planifie, ne réessaie ni ne dispatche
jamais.

## Installation

```bash
go get code.hybscloud.com/cove
```

Nécessite Go 1.27+.

## Types principaux

| Type                                                | Rôle                                                                           |
|-----------------------------------------------------|--------------------------------------------------------------------------------|
| `View[C, A]`                                        | Porteur comonadique : valeur A sous contexte ambiant C                         |
| `SuspensionView[C, A]`                              | Suspension kont appariée avec le contexte ambiant                              |
| `World`                                             | Lecture Kripke d'un type de contexte ambiant                                   |
| `StepIndex`                                         | Niveau fini d'approximation pour l'observation step-indexed                    |
| `Preorder[C]`                                       | Relation d'extension de mondes Kripke `w <= w'` avec contrôles locaux des lois |
| `Transition[C]`                                     | Arête candidate concrète entre deux mondes Kripke                              |
| `Forcing[C]` / `ForcingExpr[C]`                     | Exigence avec préordre de mondes et vérifications de monotonie                 |
| `Relation[C, A]`                                    | Relation Kripke step-indexed sur monde, indice et valeur                       |
| `IndexedView[C, A]` / `IndexedSuspensionView[C, A]` | Observations contextuelles avec crédit de pas explicite                        |
| `Req[C]`                                            | Prédicat contravariant sur C (forme fermeture) : `func(C) bool`                |
| `ReqExpr[C]`                                        | Prédicat contravariant sur C (forme défonctionnalisée)                         |
| `Rule[C]` / `RuleExpr[C]`                           | Prédicat nommé avec `Report` de diagnostic                                     |
| `Checked[C, A]` / `CheckedExpr[C, A]`               | Valeur conditionnée par une exigence                                           |
| `Guarded[C, A]` / `GuardedExpr[C, A]`               | Valeur conditionnée par une règle nommée                                       |

`Ambient` contraint les paramètres de type de contexte (C, D). `World` donne le vocabulaire Kripke à la même contrainte
structurelle, et `Focus` contraint les paramètres de type de valeur (A, B).

## Stepping contextuel

`StepWith` et `StepExprWith` évaluent une computation kont et associent sa première suspension au contexte ambiant.
Les copies de la `SuspensionView` renvoyée partagent le même pointeur `*kont.Suspension`. On peut conserver la vue en
attendant la réponse, la reprendre une fois avec `Resume` ou `ResumeWith`, ou l’abandonner avec `Discard`.

Utilisez `StepWith(ctx, m)` pour avancer directement un `Cont`. `Reify(m)` exécute le calcul jusqu’au premier retour
ou à la première opération. Un résultat terminé est réutilisable. Un résultat en suspens et ses copies partagent une
seule invocation. Tenter de l’avancer deux fois provoque un panic, y compris via `Reflect`. Pour chaque exécution,
réappliquez `Reify` au `Cont` réutilisable d’origine.

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

Utilisez la vue `sv` renvoyée par `Resume` ou `ResumeWith`. Une fois le calcul terminé, son champ `Suspension` vaut
nil et `sv.Ask()` renvoie le contexte de la vue.

cove utilise la classification de kont : une suspension nil indique la fin du calcul, et la valeur renvoyée est le résultat final. Nil est un résultat valide pour un type qui l’accepte ; une interface contenant un nil typé conserve son type dynamique. Une étape suspendue peut également renvoyer la valeur zéro de `A` : vérifiez donc la suspension pour identifier la fin. Le contexte porté est conservé.

`ObserveSuspension` associe un contexte à une suspension existante. Ce pilote vérifie le contexte courant avant
chaque opération et abandonne la suspension si l’exigence échoue. Ici, `dispatch` renvoie de façon synchrone la
réponse à l’opération en attente.

```go
canRun := func(r Runtime) bool { return r.Budget > 0 }
val, susp := cove.Step(m)
for susp != nil {
    sv, ok := cove.CheckSuspension(currentContext(), susp, canRun)
    if !ok {
        susp.Discard()
        return
    }
    val, susp = susp.Resume(dispatch(sv.Ask(), sv.Op()))
}
complete(val)
```

Pour utiliser une exigence sous forme d’expression, construire `req := cove.ExprAtom(canRun)` une seule fois et
appeler `cove.CheckSuspensionExpr(currentContext(), susp, req)` dans la boucle.

`Step` et `StepExpr` réexportent les évaluateurs de kont sans lier de contexte.

En cas de refus, le propriétaire peut conserver la suspension d’origine pour un contrôle ultérieur ou appeler `Discard`.
En cas d’`iox.ErrWouldBlock`, `takt.AdvanceSuspension` et `sess.Advance` renvoient la même suspension en attente.
Vérifiez de nouveau le contexte courant avant chaque nouvelle tentative. Protégez la condition jusqu’à l’action
autorisée, y compris en cas de réentrée d’un callback. `ResumeWith` applique le mapping après l'avancement, y compris à la fin.
`ResumeTo` vérifie une arête fournie ; il n'acquiert aucune observation fraîche et ne contrôle aucune exigence arbitraire.
L'éligibilité révocable n'est pas un forcing persistant ; `StepIndex` est un crédit logique, pas du temps. Dispatch,
avancement et libération des ressources externes ont des contrats distincts ; `Discard` ne termine pas les I/O.

Chaque condition temporelle indépendante se modélise par un type concret qui implémente `tempo.Residual`. L’appelant les
combine avec `tempo.Both`, applique les événements correspondants avec `state = state.Step(event)` et passe
`state.Observe()` à `CheckSuspension`. Le signal de disponibilité et la confirmation de fin de l’opération préalable
peuvent arriver dans l’un ou l’autre ordre. À la reprise du calcul, la génération en cours doit toujours être prête
et la confirmation doit lui correspondre.
[Exemple temporel](temporal_example_test.go) · [tempo](https://code.hybscloud.com/tempo)

## Preuve Kripke step-indexed

`cove` peut rendre explicite la lecture Kripke sans transformer le contexte en politique de scheduling. `Preorder[C]`
définit l'extension de mondes et expose les contrôles locaux `Extends`, `ReflexiveAt` et `TransitiveAt` ;
`Transition[C]`, `CheckTransition`, `CheckForcingTransition` et `CheckInvariantTransition` contrôlent des arêtes
concrètes ; `Forcing` apparie une exigence avec ce préordre, et `Relation` indexe la validité sémantique par monde,
`StepIndex` fini et valeur.

Utilisez `DiscreteWorlds` si les mondes peuvent être comparés avec `==` en Go et si chacun est égal à lui-même.
`TotalWorlds` relie toute paire de mondes. `ForceExpr` reflète `Force` pour `ReqExpr`.
`CheckRelation`, `WeakenIndexedView` et `WeakenIndexedSuspension` rendent explicites les contrôles de relation et
l'affaiblissement du crédit de pas.

```go
type RuntimeWorld struct {
    Epoch uint64
}

leq := func(w, next RuntimeWorld) bool {
    return w.Epoch <= next.Epoch
}
_ = cove.Preorder[RuntimeWorld](leq).ReflexiveAt(RuntimeWorld{Epoch: 1})
_ = cove.Preorder[RuntimeWorld](leq).TransitiveAt(RuntimeWorld{Epoch: 1}, RuntimeWorld{Epoch: 2}, RuntimeWorld{Epoch: 3})

canObserve := cove.Force(leq, func(w RuntimeWorld) bool {
    return w.Epoch > 0
})
_ = canObserve.MonotoneAt(RuntimeWorld{Epoch: 1}, RuntimeWorld{Epoch: 2})

rel := cove.Relate(leq, func(w RuntimeWorld, n cove.StepIndex, value int) bool {
    return uint64(n) <= w.Epoch && value >= 0
})
later := cove.Later(rel)
_ = later.Holds(RuntimeWorld{Epoch: 8}, 3, 4)
```

Au premier résultat ou à la première opération, `StepWithIndex` et `StepExprWithIndex` conservent tout le crédit initial.
Chaque reprise indexée consomme une unité. `ResumeTo` vérifie aussi que le monde suivant étend le monde courant.
Si le crédit est nul, cette boucle abandonne la suspension avant de traiter l'opération. À la fin du calcul,
`CheckCompletedRelation` vérifie la valeur renvoyée dans le monde final avec le crédit restant.

```go
val, sv := cove.StepExprWithIndex(RuntimeWorld{Epoch: 2}, 2, expr)
for !sv.Completed() {
    if sv.Index.IsZero() {
        sv.Discard()
        return
    }
    result := dispatch(sv.Ask(), sv.Op())
    next := sv.Ask()
    next.Epoch++
    val, sv = sv.ResumeTo(leq, result, next)
}
_ = cove.CheckCompletedRelation(sv, val, rel)
```

## Commandes

`Cmd[C, A, B]` est une commande contextuelle `View[C, A] -> B`. `Run` applique une commande à une `View` concrète ;
`ExtractCmd` est la commande identité ; `LiftCmd` relève dans le monde contextuel une transformation ne portant que sur
la valeur focalisée. Pour une vue `v`, `Compose(g, f)` évalue `g(Extend(v, f))`.

```go
cmd := cove.Compose(
    func(v cove.View[Runtime, int]) string {
        return fmt.Sprintf("budget=%d value=%d", v.Ask().Budget, v.Extract())
    },
    cove.LiftCmd[Runtime](func(n int) int { return n + 1 }),
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

La **forme de données** (`ReqExpr`) conserve la structure booléenne.

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

Utilisez `Req` pour un prédicat direct et `ReqExpr` pour conserver la structure booléenne. Les deux formes peuvent être
construites une seule fois. Si les entrées peuvent changer, réévaluez la condition avant d’agir.

Si `delivered` vérifie la présence d’un accusé de réception, `Not(delivered)` est aussi vrai tant que la livraison
reste incertaine. Une nouvelle tentative qui exige un rejet confirmé nécessite son propre prédicat.

`ReflectReq(expr)` permet d’évaluer l’expression sous forme de `Req`. `ReifyReq(req)` enveloppe un prédicat non nul
dans un seul `ExprAtom`. `Need(ctx, nil)` renvoie vrai. Utilisez `ExprTrue[C]()` pour une expression sans restriction.
`NeedExpr(ctx, ReifyReq[C](nil))` provoque un panic.

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

Pour C fixé et des callbacks purs, `View[C, A]` possède la structure de comonade d’environnement décrite plus bas. Le porteur est le produit C × A ; `Extract` (ε) projette la valeur ; `Duplicate` (δ) plonge l’observation comme valeur focalisée :

```go
v := cove.Observe(ctx, value)
v.Ask()       // contexte ambiant (π₁)
v.Extract()   // valeur focalisée (ε = π₂)
```

Transformations : `Map` (relèvement fonctoriel sur `A`), `MapContext` (relèvement fonctoriel sur `C`), `Replace` (
substituer la valeur), `WithContext` (substituer le contexte).

Les lois de `Duplicate` et `Extend` supposent un contexte stable et des fonctions pures et totales. `≈` désigne l’égalité observationnelle. [Tests Go](view_test.go)

| Loi | Équation |
| --- | --- |
| Counité | `Duplicate(v).Extract() ≈ v` |
| Identité d’extension | `Extend(v, ExtractCmd) ≈ v` |
| Associativité | `Extend(Extend(v, f), g) ≈ Extend(v, Compose(g, f))` |

`Extend` est l’opérateur d’extension contextuelle : étant donné `f: W(C, A) → B`, il relève `f` en `W(C, A) → W(C, B)`
en préservant le contexte ambiant. La composition de commandes induite avec `g: W(C, B) → D` est `g ∘ Extend(f)`.

## Position dans l’écosystème

| Paquet     | Responsabilité                                               | Aspect catégorique               |
|------------|--------------------------------------------------------------|----------------------------------|
| `kont`     | Opérations d’effets, gestionnaires, suspension et reprise    | Algèbre / Monade libre / Pli     |
| `tempo`    | Transitions et observations de l’état temporel                | Machine de Moore / Produit synchrone |
| **`cove`** | **Contexte ambiant à travers les frontières de suspension**  | **Coalgèbre / Comonade / Dépli** |
| `takt`     | Dispatch proacteur, stepping et observation du runner        | Comodèle (dual du gestionnaire)  |
| `uring`    | I/O noyau : SQE/CQE, gestion des anneaux, anneaux de buffers | —                                |
| `iox`      | Algèbre de résultats et sémantique des erreurs               | —                                |

## Structure formelle

Pour C fixé et des callbacks purs et totaux, `View[C, A]` est une comonade d’environnement (Uustalu & Vene 2008) : porteur C × A, counité ε = π₂ (`Extract`) et comultiplication δ(c, a) = (c, (c, a)) (`Duplicate`). `Extend` implémente l’extension coKleisli. Les équations expriment une égalité observationnelle avec un contexte stable ; elles n’exigent pas de valeurs ou de fonctions comparables en Go.

`Req[C]` désigne un prédicat ; `ReqExpr[C]` conserve une syntaxe booléenne dont les atomes sont des fonctions opaques. Pour des prédicats et projections purs, `Pullback` implémente l’application contravariante f* induite par f: C → D. `ExprPullback` préserve la structure booléenne : f*(p ∧ q) = f*(p) ∧ f*(q), f*(¬p) = ¬f*(p). La projection peut être appelée pour chaque atome visité.

kont fournit les calculs avec effets et les reprises ; cove transporte le contexte nécessaire pour observer et vérifier leurs frontières. L’appelant acquiert les faits, fournit la synchronisation et gère l’exécution ainsi que la durée de vie des ressources.

Les exigences composées sont évaluées de gauche à droite avec court-circuit. `All`, `Any`, `ExprAll` et `ExprAny` conservent les opérandes multiples fournis par slice ; ce stockage doit rester inchangé pendant son utilisation. Les copies de View partagent le stockage référencé. `ExprAtom` et `ReifyReq` exigent des prédicats non nil.

`ReqExpr` représente la composition booléenne sous forme de données ; les prédicats feuilles restent des fonctions Go. L’évaluation interprète cette structure. `ReifyReq` enveloppe un prédicat opaque dans un atome ; il n’extrait pas la syntaxe d’une fermeture et ne spécialise pas automatiquement l’exécution.

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

// 3. Vérifier l’exigence dans un contexte concret avant d’utiliser la valeur.
if view, ok := checked.IntoView(Caps{CanSubmit: true, HasToken: true}); ok {
	_ = view.Extract()
}
```

`Pullback` permet à un appelant d’adapter une exigence définie sur un contexte pour qu’elle s’applique sur un autre (
`Pullback(req, f)` avec `f: D → C`) ; `MapChecked` et `MapGuarded` transportent des valeurs conditionnées à travers des
transformations au niveau valeur sans re-vérifier le prédicat. La combinaison `All` + `Pullback` + `Guard` est la
manière dont les paquets en aval construisent des vérifications de capacités typées sans se coupler à un unique type de
contexte.

## Mesure des performances

Le [guide des benchmarks](BENCHMARKS.md) présente les charges comparables, les commandes de mesure et le périmètre des allocations dans votre environnement.

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

`cove` est un paquet Go pur dans ce module. Il n'a ni fichier spécifique à une plateforme ni build tag ; la seule exigence du module est Go 1.27+.

## Licence

Licence MIT. Voir [LICENSE](LICENSE) pour les détails.

©2026 [Hayabusa Cloud Co., Ltd.](https://code.hybscloud.com)
