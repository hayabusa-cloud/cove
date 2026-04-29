[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/cove.svg)](https://pkg.go.dev/code.hybscloud.com/cove)
[![Go Report Card](https://goreportcard.com/badge/github.com/hayabusa-cloud/cove)](https://goreportcard.com/report/github.com/hayabusa-cloud/cove)
[![Coverage Status](https://codecov.io/gh/hayabusa-cloud/cove/graph/badge.svg)](https://codecov.io/gh/hayabusa-cloud/cove)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[English](README.md) | [简体中文](README.zh-CN.md) | **Español** | [日本語](README.ja.md) | [Français](README.fr.md)

# cove

Capa de contexto para transportar contexto ambiental explícito a través de fronteras de suspensión
de [kont](https://code.hybscloud.com/kont) en Go.

## Descripción general

Cuando un runtime externo, como un proactor, un bucle de eventos o un dispatcher, procesa suspensiones de kont una a
una, cada operación suspendida puede necesitar estado que no forma parte de la operación en sí: presupuesto de dispatch,
capacidades del anillo, fase del protocolo, validez del grupo de buffers. Sin portadores de contexto explícitos, ese
estado acaba en mapas auxiliares ad hoc o globales implícitos que rompen la composición.

cove empareja el contexto ambiental con los valores y las suspensiones para que el contexto atraviese las fronteras de
stepping asíncrono como datos tipados y componibles.

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

El paquete es libre de políticas: transporta y verifica el contexto, pero nunca planifica, reintenta ni despacha.

## Instalación

```bash
go get code.hybscloud.com/cove
```

Requiere Go 1.26+.

## Tipos principales

| Tipo                                                | Propósito                                                                      |
|-----------------------------------------------------|--------------------------------------------------------------------------------|
| `View[C, A]`                                        | Portador comonádico: valor A bajo contexto ambiental C                         |
| `SuspensionView[C, A]`                              | Suspensión kont emparejada con contexto ambiental                              |
| `World`                                             | Lectura Kripke de un tipo de contexto ambiental                                |
| `StepIndex`                                         | Nivel finito de aproximación para observación step-indexed                     |
| `Preorder[C]`                                       | Relación de extensión de mundos Kripke `w <= w'` con chequeos locales de leyes |
| `Transition[C]`                                     | Arista candidata concreta entre dos mundos Kripke                              |
| `Forcing[C]` / `ForcingExpr[C]`                     | Requisito con preorden de mundos y comprobaciones de monotonía                 |
| `Relation[C, A]`                                    | Relación Kripke step-indexed sobre mundo, índice y valor                       |
| `IndexedView[C, A]` / `IndexedSuspensionView[C, A]` | Observaciones contextuales con crédito de pasos explícito                      |
| `Req[C]`                                            | Predicado contravariante sobre C (forma clausura): `func(C) bool`              |
| `ReqExpr[C]`                                        | Predicado contravariante sobre C (forma desfuncionalizada)                     |
| `Rule[C]` / `RuleExpr[C]`                           | Predicado nombrado con `Report` de diagnóstico                                 |
| `Checked[C, A]` / `CheckedExpr[C, A]`               | Valor controlado por un requisito                                              |
| `Guarded[C, A]` / `GuardedExpr[C, A]`               | Valor controlado por una regla nombrada                                        |

`Ambient` restringe parámetros de tipo de contexto (C, D). `World` da a la misma restricción estructural el vocabulario
Kripke, y `Focus` restringe parámetros de tipo de valor (A, B).

## Stepping contextual

`StepWith` y `StepExprWith` evalúan una computación kont y emparejan su primera suspensión con el contexto ambiental.
Cada suspensión es afín: se consume exactamente una vez, mediante `Resume`, `ResumeWith` o `Discard`.

```go
val, sv := cove.StepExprWith(ctx, expr)
for sv.Suspension != nil {
    op := sv.Op()
    result := handle(op)
    val, sv = sv.Resume(result)
}
```

`ResumeWith` hace evolucionar el contexto transportado entre pasos. Por ejemplo, puede descontar presupuesto, actualizar
capacidades o avanzar la fase del protocolo:

```go
val, sv = sv.ResumeWith(result, func(r Runtime) Runtime {
    r.Budget--
    return r
})
```

`MapContextSuspension` y `WithContextSuspension` transportan o sustituyen explícitamente el contexto llevado por una
suspensión ya observada, sin alterar la frontera de suspensión actual:

```go
sv = cove.MapContextSuspension(sv, func(r Runtime) Runtime {
    r.Budget += 4
    return r
})
sv = cove.WithContextSuspension(sv, Runtime{Budget: 16})
```

Una vez completada la computación, `sv.Suspension` pasa a ser nil, pero `sv.Ask()` sigue devolviendo el contexto
transportado.

cove reenvía el clasificador de stepping de kont, por lo que `Step`, `StepExpr`, `StepWith`, `StepExprWith`, `Resume` y
`ResumeWith` heredan la convención de finalización con nil de kont: la finalización se observa mediante una suspensión
nil, en cuyo caso el valor final sigue la convención de kont (un valor nil denota el valor cero de `A`). Por tanto,
las computaciones cuyo tipo de resultado sea un tipo que admita nil (puntero, interfaz, mapa, slice, canal o función)
no pueden usar nil como valor final significativo; envuelve nil en un tipo suma o testigo explícito cuando esa
distinción sea relevante. El contexto del portador no se ve afectado.

`ObserveSuspension` adjunta contexto ambiental a una `kont.Suspension` existente sin realizar ninguna comprobación de
requisitos. `CheckSuspension` y `CheckSuspensionExpr` proporcionan las variantes condicionadas:

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

`Step` y `StepExpr` reexportan los evaluadores de kont sin vincular contexto.

## Evidencia Kripke step-indexed

`cove` puede hacer explícita la lectura Kripke sin convertir el contexto en política de scheduling. `Preorder[C]` define
la extensión de mundos y expone chequeos locales `Extends`, `ReflexiveAt` y `TransitiveAt`; `Transition[C]`,
`CheckTransition`, `CheckForcingTransition` y `CheckInvariantTransition` comprueban aristas concretas; `Forcing`
empareja un requisito con ese preorden, y `Relation` indexa la validez semántica por mundo, `StepIndex` finito y valor.

Use `DiscreteWorlds` para mundos relacionados solo por igualdad y `TotalWorlds` para un preorden que relaciona todo par
de mundos. `ForceExpr` refleja `Force` para `ReqExpr`; `CheckRelation`, `WeakenIndexedView` y `WeakenIndexedSuspension`
hacen explícitos los chequeos de relación y el debilitamiento del crédito de pasos.

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

Para prefijos operacionales, `StepWithIndex` y `StepExprWithIndex` emparejan un `SuspensionView` con fuel explícito.
Cada `Resume` indexado consume una unidad de crédito de paso, haciendo visible el descenso de las observaciones finitas.
Use `ResumeTo` cuando el contexto sucesor deba verificarse como extensión de mundo Kripke; `CheckCompletedRelation`
comprueba la relación final en una frontera indexada completada:

```go
_, sv := cove.StepExprWithIndex(RuntimeWorld{Epoch: 2}, 2, expr)
var val int
for sv.Extract() != nil {
    result := dispatch(sv.Ask(), sv.Op())
    next := sv.Ask()
    next.Epoch++
    val, sv = sv.ResumeTo(leq, result, next)
}
_ = cove.CheckCompletedRelation(sv, val, rel)
```

## Comandos

`Cmd[C, A, B]` es un comando contextual `View[C, A] -> B`. `Run` aplica un comando a un `View` concreto; `ExtractCmd` es
el comando identidad; `LiftCmd` eleva al mundo contextual una transformación centrada solo en el valor enfocado;
`Compose` compone comandos mediante `Extend`.

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

## Requisitos

Los requisitos son predicados sobre el contexto ambiental, disponibles tanto en forma clausura como en forma datos.
Ambas formas admiten `All`, `Any` y `Not`, con `True` y `False` como elementos neutros de la conjunción y la disyunción.

**Forma clausura** (`Req`): directa y concisa.

```go
req := cove.All(
    cove.Req[Runtime](func(r Runtime) bool { return r.Budget > 0 }),
    cove.Req[Runtime](func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.Need(runtime, req)
```

**Forma datos** (`ReqExpr`): estructura booleana componible que evita la asignación de clausuras durante la composición.

```go
expr := cove.ExprAll(
    cove.ExprAtom(func(r Runtime) bool { return r.Budget > 0 }),
    cove.ExprAtom(func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.NeedExpr(runtime, expr)
```

Combinadores: `All`/`ExprAll` (conjunción ∧), `Any`/`ExprAny` (disyunción ∨), `Not`/`ExprNot` (negación ¬), `True`/`ExprTrue` (⊤), `False`/`ExprFalse` (⊥).

`Pullback` y `ExprPullback` transportan requisitos de forma contravariante a lo largo de una proyección de contexto.
Dada `f: C → D`, `Pullback(req, f)` lleva `Req[D]` a `Req[C]`. `ExprPullback` preserva la estructura booleana y se
distribuye sobre `ExprNot`, `ExprAll` y `ExprAny`.

### Elegir entre Req y ReqExpr

Use `Req` para predicados puntuales y ad hoc. Use `ReqExpr` cuando componga muchos requisitos, o cuando importe la
asignación de clausuras en tiempo de construcción.

Los helpers de puente mantienen alineadas ambas formas: `ReifyReq` envuelve un requisito en forma clausura como un
`ExprAtom`, por lo que `ReifyReq(nil)` es inválido y entra en pánico al evaluarse; `ReflectReq` evalúa un requisito en
forma Expr a través de una clausura.

## Valores controlados

`Guard` empareja un valor con un requisito; `IntoView` devuelve un `View` sólo cuando el contexto ambiental satisface
dicho requisito:

```go
checked := cove.Guard(canDispatch, payload)
if view, ok := checked.IntoView(runtime); ok {
    _ = view.Extract()
}
```

`GuardRule` añade diagnósticos mediante reglas nombradas y `Report`:

```go
rule := cove.Require("budget", func(r Runtime) bool { return r.Budget > 0 })
guarded := cove.GuardRule(rule, payload)
if view, report := guarded.IntoView(runtime); report.OK() {
    _ = view.Extract()
}
```

`CheckRules` y `CheckRulesExpr` evalúan varias reglas en orden, deteniéndose en el primer fallo:

```go
report := cove.CheckRules(runtime, budgetRule, permRule)
if !report.OK() {
    // report.Failed nombra la primera regla que no se cumplió
}
```

La ruta de diagnóstico de la forma Expr tiene la misma forma a través de `CheckRulesExpr` y `GuardRuleExpr`:

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

Equivalentes en forma Expr: `GuardExpr`, `GuardRuleExpr`, `CheckedExpr`, `GuardedExpr`. Helpers de map y pullback:
`MapChecked`, `MapGuarded`, `PullbackChecked`, `PullbackGuarded` (y sus variantes Expr).

## Operaciones de View

`View[C, A]` es una comónada sobre la categoría de tipos Go. El portador es el producto C × A; `Extract` (ε) proyecta el valor; `Duplicate` (δ) incrusta la observación como valor enfocado:

```go
v := cove.Observe(ctx, value)
v.Ask()       // contexto ambiental (π₁)
v.Extract()   // valor enfocado (ε = π₂)
```

Transformaciones: `Map` (elevación funtorial sobre `A`), `MapContext` (elevación funtorial sobre `C`), `Replace` (
sustituir el valor), `WithContext` (sustituir el contexto).

`Duplicate` y `Extend` satisfacen las leyes comonádicas:

```go
// Ley de counidad: ε ∘ δ = id
cove.Duplicate(v).Extract() == v

// Identidad de extensión: extend(ε) = id
cove.Extend(v, func(w cove.View[C, A]) A { return w.Extract() }) == v

// Asociatividad: extend(f) ∘ extend(g) = extend(f ∘ extend(g))
```

`Extend` es el operador de extensión contextual: dado `f: W(C, A) → B`, eleva `f` a `W(C, A) → W(C, B)` preservando el
contexto ambiental. La composición de comandos inducida con `g: W(C, B) → D` es `g ∘ Extend(f)`.

## Posición en el ecosistema

| Paquete    | Responsabilidad                                                 | Aspecto categórico                    |
|------------|-----------------------------------------------------------------|---------------------------------------|
| `kont`     | Operaciones de efectos, manejadores, suspensión y reanudación   | Álgebra / Mónada libre / Pliegue      |
| **`cove`** | **Contexto ambiental a través de fronteras de suspensión**      | **Coálgebra / Comónada / Despliegue** |
| `takt`     | Dispatch proactor, stepping y observación del runner            | Comodelo (dual del manejador)         |
| `uring`    | I/O del kernel: SQE/CQE, gestión de anillos, anillos de buffers | —                                     |
| `iox`      | Álgebra de resultados y semántica de errores                    | —                                     |

## Estructura formal

`View[C, A]` es una comónada de entorno (Uustalu & Vene 2008) con portador C × A, counidad ε = π₂ (Extract) y
comultiplicación δ(c, a) = (c, (c, a)) (Duplicate). `Extend` implementa la extensión contextual (extensión coKleisli en
la literatura categórica).

`Req[C]` y `ReqExpr[C]` son objetos en el álgebra booleana de predicados sobre C. `Pullback` implementa el funtor contravariante f* inducido por un morfismo f: C → D, mapeando predicados sobre D a predicados sobre C. `ExprPullback` preserva la estructura del álgebra booleana: f*(p ∧ q) = f*(p) ∧ f*(q), f*(¬p) = ¬f*(p).

kont modela efectos algebraicos (mónada libre, manejadores, pliegue); cove modela coefectos (comónada, requisitos,
despliegue). Los dos son duales categóricos en la frontera de suspensión: kont describe qué le hace la computación al
mundo, mientras que cove declara qué necesita la computación del mundo.

Leído bajo la óptica de los efectos modales, la forma del portador puede leerse como el nombre de lo que ocurre con el
contexto ambiente: kont (C → D) lo reemplaza, cove (W(C, A) → B) lo maneja de forma relativa, y el paso puro lo
preserva. La estructura de efectos viaja así sobre el portador, no sobre un «color» de función, lo que mantiene a cove
libre de políticas.

`ReqExpr` desfuncionaliza el álgebra booleana en una unión etiquetada, es decir, el álgebra inicial del funtor de firma
booleana, de modo que la estructura de un requisito sea un dato inspeccionable en lugar de una clausura opaca (Reynolds
1972).

## Patrones prácticos

Una computación contextual completa suele avanzar en tres etapas: declarar requisitos, condicionar valores sobre ellos
y, por último, observar el resultado bajo un contexto concreto.

```go
// 1. Declarar un requisito sobre el contexto ambiente.
type Caps struct{ CanSubmit, HasToken bool }
req := cove.All(
    func(c Caps) bool { return c.CanSubmit },
    func(c Caps) bool { return c.HasToken },
)

// 2. Condicionar un valor con ese requisito. Esto produce un Checked[Caps, T].
checked := cove.Guard(req, payload)

// 3. Reificarlo a un Expr para poder avanzar bajo distintos contextos ambientes.
expr := cove.ReifyReq(req)
ok   := cove.NeedExpr(Caps{CanSubmit: true, HasToken: true}, expr)
_    = checked
_    = ok
```

`Pullback` permite al llamador adaptar un requisito definido sobre un contexto para que opere sobre otro (
`Pullback(req, f)` con `f: D → C`); `MapChecked` y `MapGuarded` transportan valores condicionados a través de
transformaciones a nivel de valor sin volver a comprobar el predicado. La combinación `All` + `Pullback` + `Guard` es la
forma en que los paquetes río abajo construyen comprobaciones de capacidades tipadas sin acoplarse a un único tipo de
contexto.

## Referencias

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

## Compatibilidad de plataforma

`cove` es un paquete puro de Go en este módulo. No tiene archivos específicos de plataforma ni build tags; el requisito del módulo es Go 1.26+.

## Licencia

Licencia MIT. Consulte [LICENSE](LICENSE) para más detalles.

©2026 [Hayabusa Cloud Co., Ltd.](https://code.hybscloud.com)
