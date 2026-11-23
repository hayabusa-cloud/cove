[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/cove.svg)](https://pkg.go.dev/code.hybscloud.com/cove)
[![Benchmarks](https://github.com/hayabusa-cloud/cove/actions/workflows/benchmarks.yml/badge.svg)](https://github.com/hayabusa-cloud/cove/actions/workflows/benchmarks.yml)
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
for sv.Suspension != nil && sv.Ask().Budget > 0 {
    result := dispatch(sv.Ask(), sv.Op())
    _, sv = sv.ResumeWith(result, func(r Runtime) Runtime {
        r.Budget--
        return r
    })
}
```

Si `sv.Suspension != nil` al salir del bucle, conserva `sv` y repón su presupuesto antes del siguiente despacho.

El paquete es libre de políticas: transporta y verifica el contexto, pero nunca planifica, reintenta ni despacha.

## Instalación

```bash
go get code.hybscloud.com/cove
```

Requiere Go 1.27+.

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
Las copias de la `SuspensionView` devuelta comparten el mismo puntero `*kont.Suspension`. La vista puede conservarse
mientras se espera la respuesta. Se reanuda una vez con `Resume` o `ResumeWith`, o se abandona con `Discard`.

Use `StepWith(ctx, m)` para avanzar un `Cont` directamente. `Reify(m)` ejecuta el cálculo hasta el primer retorno u
operación. Un resultado completado es reutilizable. Un resultado pendiente y sus copias comparten una sola invocación.
Avanzarlo dos veces provoca un pánico, incluso a través de `Reflect`. Para cada ejecución, vuelva a aplicar `Reify`
al `Cont` reutilizable original.

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

cove utiliza la clasificación de kont: una suspensión nil indica la finalización y el valor devuelto es el resultado final. Nil es un resultado válido para los tipos que lo admiten; una interfaz que contiene un nil tipado conserva su tipo dinámico. Un paso suspendido también puede devolver el valor cero de `A`, por lo que debe comprobarse la suspensión para identificar la finalización. El contexto se conserva.

`ObserveSuspension` vincula un contexto a una suspensión existente. Este controlador comprueba el contexto actual antes
de cada despacho y descarta la suspensión si no se cumple el requisito. Aquí `dispatch` devuelve de forma síncrona la
respuesta de la operación pendiente.

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

Para usar un requisito en forma de expresión, se construye `req := cove.ExprAtom(canRun)` una vez y se llama a
`cove.CheckSuspensionExpr(currentContext(), susp, req)` dentro del bucle.

`Step` y `StepExpr` reexportan los evaluadores de kont sin vincular contexto.

Si se rechaza la operación, el propietario puede conservar la suspensión original para otra comprobación o llamar a
`Discard`. Ante `iox.ErrWouldBlock`, `takt.AdvanceSuspension` y `sess.Advance` devuelven la misma suspensión pendiente.
Vuelve a comprobar el contexto actual antes de reintentar. Protege la condición hasta la acción autorizada, incluida la
reentrada de callbacks. `ResumeWith` aplica el mapeador después del avance, también al finalizar.
`ResumeTo` comprueba una arista suministrada; no obtiene observaciones nuevas ni comprueba requisitos arbitrarios.
La elegibilidad revocable no es forcing persistente y `StepIndex` es crédito lógico, no tiempo. Despacho, avance y
liberación de recursos externos tienen contratos separados; `Discard` no completa I/O.

Cada condición temporal independiente se modela con un tipo concreto que implementa `tempo.Residual`. El llamador las
combina con `tempo.Both`, aplica los eventos correspondientes con `state = state.Step(event)` y pasa `state.Observe()` a
`CheckSuspension`. La señal de disponibilidad y la confirmación de que terminó la operación previa pueden llegar en
cualquier orden. Para reanudar la computación, la generación actual debe seguir lista y la confirmación debe
corresponder a esa generación.
[Ejemplo temporal](temporal_example_test.go) · [tempo](https://code.hybscloud.com/tempo)

## Evidencia Kripke step-indexed

`cove` puede hacer explícita la lectura Kripke sin convertir el contexto en política de scheduling. `Preorder[C]` define
la extensión de mundos y expone chequeos locales `Extends`, `ReflexiveAt` y `TransitiveAt`; `Transition[C]`,
`CheckTransition`, `CheckForcingTransition` y `CheckInvariantTransition` comprueban aristas concretas; `Forcing`
empareja un requisito con ese preorden, y `Relation` indexa la validez semántica por mundo, `StepIndex` finito y valor.

Use `DiscreteWorlds` cuando los mundos se puedan comparar con `==` en Go y cada uno sea igual a sí mismo.
`TotalWorlds` relaciona cualquier par de mundos. `ForceExpr` refleja `Force` para `ReqExpr`.
`CheckRelation`, `WeakenIndexedView` y `WeakenIndexedSuspension` hacen explícitos los chequeos de relación y el
debilitamiento del crédito de pasos.

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

Al obtener el primer resultado u operación, `StepWithIndex` y `StepExprWithIndex` conservan todo el crédito inicial.
Cada reanudación indexada consume una unidad. `ResumeTo` también comprueba que el mundo sucesor extienda al actual.
Si el crédito es cero, este bucle descarta la suspensión pendiente antes de despachar la operación. Al terminar,
`CheckCompletedRelation` comprueba el valor devuelto en el mundo final con el crédito restante.

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

## Comandos

`Cmd[C, A, B]` es un comando contextual `View[C, A] -> B`. `Run` aplica un comando a un `View` concreto; `ExtractCmd` es
el comando identidad y `LiftCmd` eleva al mundo contextual una transformación centrada solo en el valor enfocado.
Para una vista `v`, `Compose(g, f)` evalúa `g(Extend(v, f))`.

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

La **forma de datos** (`ReqExpr`) conserva la estructura booleana.

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

Use `Req` para un predicado directo y `ReqExpr` para conservar la estructura booleana. Ambos pueden construirse una
sola vez. Si los datos de entrada pueden cambiar, vuelva a evaluar el requisito antes de actuar.

Si `delivered` comprueba el acuse de recibo, `Not(delivered)` también se cumple mientras se desconoce si hubo entrega.
Un reintento que exige un rechazo confirmado necesita su propio predicado.

`ReflectReq(expr)` permite evaluar la expresión como un `Req`. `ReifyReq(req)` envuelve un predicado no nulo en un
solo `ExprAtom`. `Need(ctx, nil)` devuelve verdadero. Use `ExprTrue[C]()` para una expresión sin restricciones.
`NeedExpr(ctx, ReifyReq[C](nil))` provoca un pánico.

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

Para C fijo y callbacks puros, `View[C, A]` tiene la estructura de comónada de entorno descrita más adelante. El portador es el producto C × A; `Extract` (ε) proyecta el valor; `Duplicate` (δ) incrusta la observación como valor enfocado:

```go
v := cove.Observe(ctx, value)
v.Ask()       // contexto ambiental (π₁)
v.Extract()   // valor enfocado (ε = π₂)
```

Transformaciones: `Map` (elevación funtorial sobre `A`), `MapContext` (elevación funtorial sobre `C`), `Replace` (
sustituir el valor), `WithContext` (sustituir el contexto).

Las leyes de `Duplicate` y `Extend` suponen un contexto estable y funciones puras y totales. `≈` denota igualdad observacional. [Pruebas en Go](view_test.go)

| Ley | Ecuación |
| --- | --- |
| Counidad | `Duplicate(v).Extract() ≈ v` |
| Identidad de extensión | `Extend(v, ExtractCmd) ≈ v` |
| Asociatividad | `Extend(Extend(v, f), g) ≈ Extend(v, Compose(g, f))` |

`Extend` es el operador de extensión contextual: dado `f: W(C, A) → B`, eleva `f` a `W(C, A) → W(C, B)` preservando el
contexto ambiental. La composición de comandos inducida con `g: W(C, B) → D` es `g ∘ Extend(f)`.

## Posición en el ecosistema

| Paquete    | Responsabilidad                                                 | Aspecto categórico                    |
|------------|-----------------------------------------------------------------|---------------------------------------|
| `kont`     | Operaciones de efectos, manejadores, suspensión y reanudación   | Álgebra / Mónada libre / Pliegue      |
| `tempo`    | Transiciones y observaciones del estado temporal                | Máquina de Moore / Producto síncrono  |
| **`cove`** | **Contexto ambiental a través de fronteras de suspensión**      | **Coálgebra / Comónada / Despliegue** |
| `takt`     | Dispatch proactor, stepping y observación del runner            | Comodelo (dual del manejador)         |
| `uring`    | I/O del kernel: SQE/CQE, gestión de anillos, anillos de buffers | —                                     |
| `iox`      | Álgebra de resultados y semántica de errores                    | —                                     |

## Estructura formal

Para C fijo y callbacks puros y totales, `View[C, A]` es una comónada de entorno (Uustalu & Vene 2008): portador C × A, counidad ε = π₂ (`Extract`) y comultiplicación δ(c, a) = (c, (c, a)) (`Duplicate`). `Extend` implementa la extensión coKleisli. Las ecuaciones expresan igualdad observacional con contexto estable; no requieren valores ni funciones comparables en Go.

`Req[C]` denota un predicado; `ReqExpr[C]` almacena sintaxis booleana con funciones opacas como átomos. Para predicados y proyecciones puros, `Pullback` implementa la aplicación contravariante f* inducida por f: C → D. `ExprPullback` preserva la estructura booleana: f*(p ∧ q) = f*(p) ∧ f*(q), f*(¬p) = ¬f*(p). Puede aplicar la proyección a cada átomo visitado.

kont proporciona computaciones con efectos y reanudaciones; cove transporta el contexto necesario para observar y comprobar sus fronteras. El llamador obtiene los hechos, proporciona la sincronización y controla la ejecución y la vida útil de los recursos.

Los requisitos compuestos se evalúan de izquierda a derecha con cortocircuito. `All`, `Any`, `ExprAll` y `ExprAny` retienen los múltiples operandos recibidos en un slice; ese almacenamiento debe permanecer sin cambios mientras se utilice. Las copias de View comparten el almacenamiento referenciado. `ExprAtom` y `ReifyReq` requieren predicados no nil.

`ReqExpr` representa la composición booleana como datos; los predicados hoja siguen siendo funciones de Go. La evaluación interpreta esa estructura. `ReifyReq` envuelve un predicado opaco como átomo; no extrae sintaxis de una clausura ni especializa automáticamente la ejecución.

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

// 3. Comprobar el requisito en un contexto concreto antes de usar el valor.
if view, ok := checked.IntoView(Caps{CanSubmit: true, HasToken: true}); ok {
	_ = view.Extract()
}
```

`Pullback` permite al llamador adaptar un requisito definido sobre un contexto para que opere sobre otro (
`Pullback(req, f)` con `f: D → C`); `MapChecked` y `MapGuarded` transportan valores condicionados a través de
transformaciones a nivel de valor sin volver a comprobar el predicado. La combinación `All` + `Pullback` + `Guard` es la
forma en que los paquetes río abajo construyen comprobaciones de capacidades tipadas sin acoplarse a un único tipo de
contexto.

## Medición del rendimiento

La [guía de benchmarks](BENCHMARKS.md) describe cargas comparables, comandos de medición y límites de asignación para su entorno.

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

`cove` es un paquete puro de Go en este módulo. No tiene archivos específicos de plataforma ni build tags; el requisito del módulo es Go 1.27+.

## Licencia

Licencia MIT. Consulte [LICENSE](LICENSE) para más detalles.

©2026 [Hayabusa Cloud Co., Ltd.](https://code.hybscloud.com)
