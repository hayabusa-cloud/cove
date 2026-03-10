[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/cove.svg)](https://pkg.go.dev/code.hybscloud.com/cove)
[![Go Report Card](https://goreportcard.com/badge/github.com/hayabusa-cloud/cove)](https://goreportcard.com/report/github.com/hayabusa-cloud/cove)
[![Coverage Status](https://codecov.io/gh/hayabusa-cloud/cove/graph/badge.svg)](https://codecov.io/gh/hayabusa-cloud/cove)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[English](README.md) | [简体中文](README.zh-CN.md) | **Español** | [日本語](README.ja.md) | [Français](README.fr.md)

# cove

Capa de contexto coalgébrica para suspensiones de [kont](https://code.hybscloud.com/kont) en Go, el dual coefectual de los efectos algebraicos de kont.

## Descripción general

Cuando un runtime externo, como un proactor, bucle de eventos o dispatcher, procesa suspensiones de kont una a una, cada operación suspendida puede necesitar estado que no forma parte de la operación en sí: presupuesto de dispatch, capacidades del anillo, fase del protocolo, validez del grupo de buffers. Sin portadores de contexto explícitos, este estado acaba en mapas auxiliares ad hoc o globales implícitos que rompen la composición.

cove empareja el contexto ambiental con valores y suspensiones para que el contexto viaje a través de fronteras de stepping asíncrono como datos tipados y componibles.

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

El paquete es libre de políticas: transporta y verifica contexto pero nunca planifica, reintenta ni despacha.

## Instalación

```bash
go get code.hybscloud.com/cove
```

Requiere Go 1.26+.

## Tipos principales

| Tipo | Propósito |
|------|-----------|
| `View[C, A]` | Portador comonádico: valor A bajo contexto ambiental C |
| `SuspensionView[C, A]` | Suspensión kont emparejada con contexto ambiental |
| `Req[C]` | Funtor predicado contravariante sobre C (forma clausura): `func(C) bool` |
| `ReqExpr[C]` | Funtor predicado contravariante sobre C (forma desfuncionalizada) |
| `Rule[C]` / `RuleExpr[C]` | Predicado nombrado con `Report` de diagnóstico |
| `Checked[C, A]` / `CheckedExpr[C, A]` | Valor controlado por un requisito |
| `Guarded[C, A]` / `GuardedExpr[C, A]` | Valor controlado por una regla nombrada |

`Ambient` restringe parámetros de tipo de contexto (C, D). `Focus` restringe parámetros de tipo de valor (A, B).

## Stepping contextual

`StepWith` y `StepExprWith` evalúan una computación kont y emparejan la primera suspensión con contexto ambiental. Cada suspensión es afín: se consume exactamente una vez mediante `Resume`, `ResumeWith` o `Discard`.

```go
val, sv := cove.StepExprWith(ctx, expr)
for sv.Suspension != nil {
    op := sv.Op()
    result := handle(op)
    val, sv = sv.Resume(result)
}
```

`ResumeWith` actualiza el contexto entre pasos, por ejemplo para descontar presupuesto, actualizar capacidades o cambiar de fase:

```go
val, sv = sv.ResumeWith(result, func(r Runtime) Runtime {
    r.Budget--
    return r
})
```

`ObserveSuspension` adjunta contexto ambiental a una `kont.Suspension` existente sin comprobar requisitos. `CheckSuspension` y `CheckSuspensionExpr` añaden las variantes condicionadas:

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

## Requisitos

Los requisitos son predicados sobre el contexto, disponibles en forma clausura y forma datos. Ambas admiten `All`/`Any`/`Not`, con `True` y `False` como elementos neutros de la conjunción y la disyunción.

**Forma clausura** (`Req`) — directa y concisa:

```go
req := cove.All(
    cove.Req[Runtime](func(r Runtime) bool { return r.Budget > 0 }),
    cove.Req[Runtime](func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.Need(runtime, req)
```

**Forma datos** (`ReqExpr`) — estructura booleana componible, evita la asignación de clausuras en la composición:

```go
expr := cove.ExprAll(
    cove.ExprAtom(func(r Runtime) bool { return r.Budget > 0 }),
    cove.ExprAtom(func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.NeedExpr(runtime, expr)
```

Combinadores: `All`/`ExprAll` (conjunción ∧), `Any`/`ExprAny` (disyunción ∨), `Not`/`ExprNot` (negación ¬), `True`/`ExprTrue` (⊤), `False`/`ExprFalse` (⊥).

`Pullback` y `ExprPullback` transportan requisitos de forma contravariante a lo largo de una proyección de contexto. Dada f: C → D, `Pullback(req, f)` lleva `Req[D]` a `Req[C]`. `ExprPullback` preserva la estructura booleana y distribuye sobre `ExprNot`, `ExprAll` y `ExprAny`.

### Elegir entre Req y ReqExpr

Use `Req` para predicados puntuales. Use `ReqExpr` al componer muchos requisitos o cuando importa la asignación de clausuras en tiempo de construcción.

Los helpers de puente mantienen alineadas ambas formas: `ReifyReq` envuelve un requisito en forma clausura como `ExprAtom`, por lo que `ReifyReq(nil)` es inválido y entra en pánico al evaluarse; `ReflectReq` evalúa un requisito del mundo Expr mediante una clausura.

## Valores controlados

`Guard` empareja un valor con un requisito; `IntoView` devuelve un `View` solo cuando el contexto lo satisface:

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

`CheckRules` y `CheckRulesExpr` evalúan múltiples reglas, deteniéndose en el primer fallo:

```go
report := cove.CheckRules(runtime, budgetRule, permRule)
if !report.OK() {
    // report.Failed nombra la primera regla que no se cumplió
}
```

La ruta de diagnóstico del mundo Expr sigue la misma forma con `CheckRulesExpr` y `GuardRuleExpr`:

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

Equivalentes del mundo Expr: `GuardExpr`, `GuardRuleExpr`, `CheckedExpr`, `GuardedExpr`. Helpers de map y pullback: `MapChecked`, `MapGuarded`, `PullbackChecked`, `PullbackGuarded` (y sus variantes Expr).

## Operaciones de View

`View[C, A]` es una comónada sobre la categoría de tipos Go. El portador es el producto C × A; `Extract` (ε) proyecta el valor; `Duplicate` (δ) incrusta la observación como valor enfocado:

```go
v := cove.Observe(ctx, value)
v.Ask()       // contexto ambiental (π₁)
v.Extract()   // valor enfocado (ε = π₂)
```

Transformaciones: `Map` (elevación funtorial sobre A), `MapContext` (elevación funtorial sobre C), `Replace` (sustituir valor), `WithContext` (sustituir contexto).

`Duplicate` y `Extend` satisfacen las leyes comonádicas:

```go
// Ley de counidad: ε ∘ δ = id
cove.Duplicate(v).Extract() == v

// Identidad coKleisli: extend(ε) = id
cove.Extend(v, func(w cove.View[C, A]) A { return w.Extract() }) == v

// Asociatividad: extend(f) ∘ extend(g) = extend(f ∘ extend(g))
```

`Extend` es extensión coKleisli: dado f: W(C,A) → B, eleva f a W(C,A) → W(C,B) preservando el contexto ambiental. La composición coKleisli inducida con g: W(C,B) → D es g ∘ Extend(f).

## Posición en el ecosistema

| Paquete | Responsabilidad | Aspecto categórico |
|---------|-----------------|--------------------|
| `kont` | Operaciones de efectos, manejadores, suspensión y reanudación | Álgebra / Mónada libre / Pliegue |
| **`cove`** | **Contexto ambiental a través de fronteras de suspensión** | **Coálgebra / Comónada / Despliegue** |
| `takt` | Dispatch proactor, clasificación de resultados, bucle de eventos | Comodelo (dual del manejador) |
| `uring` | I/O del kernel: SQE/CQE, gestión de anillos, anillos de buffers | — |
| `iox` | Álgebra de resultados y semántica de errores | — |

## Estructura formal

`View[C, A]` es una comónada de entorno (Uustalu & Vene 2008) con portador C × A, counidad ε = π₂ (Extract) y comultiplicación δ(c, a) = (c, (c, a)) (Duplicate). `Extend` implementa la extensión coKleisli.

`Req[C]` y `ReqExpr[C]` son objetos en el álgebra booleana de predicados sobre C. `Pullback` implementa el funtor contravariante f* inducido por un morfismo f: C → D, mapeando predicados sobre D a predicados sobre C. `ExprPullback` preserva la estructura del álgebra booleana: f*(p ∧ q) = f*(p) ∧ f*(q), f*(¬p) = ¬f*(p).

kont modela efectos algebraicos (mónada libre, manejadores, pliegue). cove modela coefectos (comónada, requisitos, despliegue). Los dos son duales categóricos en la frontera de suspensión: kont pregunta qué hace la computación al mundo; cove declara qué necesita la computación del mundo.

`ReqExpr` desfuncionaliza el álgebra booleana en una unión etiquetada — el álgebra inicial del funtor de firma booleana — de modo que la estructura de requisitos sea datos inspeccionables en lugar de clausuras opacas (Reynolds 1972).

## References

- Uustalu, T. & Vene, V. "Comonadic Notions of Computation." *CMCS 2008*, pp. 263–284. https://doi.org/10.1016/j.entcs.2008.05.029
- Petricek, T., Orchard, D. & Mycroft, A. "Coeffects: A Calculus of Context-Dependent Computation." *ICFP 2014*, pp. 123–135. https://doi.org/10.1145/2628136.2628160
- Reynolds, J.C. "Definitional Interpreters for Higher-Order Programming Languages." *ACM '72*, pp. 717–740. https://doi.org/10.1145/800194.805852

## Compatibilidad de plataforma

`cove` es un paquete puro de Go en este módulo. No tiene archivos específicos de plataforma ni build tags; el requisito del módulo es Go 1.26+.

## Licencia

Licencia MIT. Consulte [LICENSE](LICENSE) para más detalles.

©2026 [Hayabusa Cloud Co., Ltd.](https://code.hybscloud.com)
