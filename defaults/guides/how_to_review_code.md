# Guía de Code Review profunda (portable entre repos)

Pega esta guía a cualquier agente/modelo capaz de leer código y ejecutar shell.
Sirve para revisar PRs de forma profunda y **fiable**: cada hallazgo se verifica
contra el código real antes de escribirse.

**Regla de oro:** una review sin verificar contra el código es una alucinación
con formato bonito. Todo el flujo existe para evitar eso. Se revisa profundo, no
en superficie.

Idioma: **español**. Tono: llano, entendible por no-expertos, pero **corto** —
nada de relleno. Cada hallazgo = qué pasa (1-3 frases) + **Fix:** concreto.
Brevedad fuerte, pero cubriendo **todas** las severidades (corto ≠ incompleto:
las menores también van).

---

## 0. Contexto que necesitas antes de empezar

Rellena esto para el repo actual (varía entre proyectos):

- **Gestor de PRs / API:** `gh` (GitHub).
- **Rama base de cada PR:** NO asumas `main`. Una PR puede basarse en otra rama
  de feature. Compruébalo: `gh pr view <N> --json baseRefName,headRefName`.
  Revisa cada PR **contra su base real**, sin re-revisar trabajo ya mergeado.
- **Qué revisar:** normalmente el **delta** desde la última review (los commits
  nuevos: `git log <base>..<head>`), no todo el repo.
- **Convenciones del repo** que sirven de "verdad" para detectar desviaciones:
  guards de auth, helpers compartidos, patrón de i18n, migraciones, estilo de
  comentarios, etc. (léelas de `AGENTS.md` / `CLAUDE.md` / `README` / `.ctx/`).
- **Opiniones duras del repo a respetar** (rellena las tuyas): p. ej. modelos
  aprobados vs. proscritos, librerías vetadas, patrones prohibidos. Si el diff
  reintroduce algo proscrito por el repo, es hallazgo.

---

## 1. Qué cazar (criterios de fondo)

Esto es **en qué consiste** la review: lo que buscas al leer el diff. Todo lo
demás (armar la lista, verificar) es el mecanismo para aplicar estos criterios.

- **Coherencia con el código previo = criterio #1.** Compara cada cambio contra
  la convención existente del repo. Una desviación del patrón establecido es
  hallazgo, salvo que sea deliberada y esté documentada (entonces se marca como
  tal, no como bug).
- **Impacto concreto y medible**, no teórico: coste (€/llamadas), escala (N+1,
  seq-scan a X filas), contrato de salida roto, corrupción de datos, agujero de
  auth. Si no sabes decir a quién/cuánto le duele, probablemente no es hallazgo.
- **Sobre-ingeniería e indirecciones inútiles:** getters que solo devuelven una
  constante, wrappers de una línea, capas que no aportan. Márcalos.
- **Configurabilidad falsa:** flags/vars que fingen ser configurables pero están
  hardcodeadas o ignoradas ("esta constante es mentira"). Márcalo.
- **Dead code** y fuentes de verdad dobles.
- **Ruido en comentarios:** docstrings obvios o sobre-explicados, comentarios que
  repiten el código. Solo se justifican los *load-bearing*. Rubric completo en la
  **sección 4**.
- **Churn no relacionado** que infla el diff (formateo masivo, unicode, imports
  reordenados sin razón). Sepáralo del cambio real.
- **Higiene de repo y seguridad:** ficheros mal ubicados, basura commiteada,
  secretos en `.env.example` o en el código (🚨 alta prioridad, siempre visible).

---

## 2. Identificar los puntos a revisar

Aplicando los criterios de la sección 1 al delta, arma la **lista de puntos
candidatos**. Salen de:

- Los **commits nuevos** del delta (`git log`, `git show`).
- Leer el **diff** y el código tocado.

Cada punto es un hallazgo potencial concreto ("X no valida Y en `fichero.ts:42`"),
no una dimensión abstracta. Piensa también en **higiene de repo** como puntos:
archivos mal ubicados (datasets/logs colados en `docs/`), ficheros basura
commiteados, secretos en `.env.example` (esto último = alta prioridad).

---

## 3. Verificar cada punto: auto-refutación adversarial

Verificar y re-revisar cada punto candidato **es parte de la review**, no un paso
aparte: un punto sin comprobar contra el código real no es un hallazgo, es una
sospecha. Por cada punto:

1. Toma **el punto**.
2. Intenta **refutarlo** contra el código real (mandato adversarial: "demuestra
   que esto NO es un problema"). Usa `grep` / `git` / leer el fichero de verdad.
3. Emite un veredicto: **confirmado** (con `archivo:línea` y evidencia),
   **descartado** (por qué: se corrigió, nunca existió, otra capa lo tapa), o
   **matizado** (es una desviación deliberada y documentada, no un bug).

**Para escalar**, esto se paraleliza trivialmente: **un agente por punto**, cada
uno auto-verificando el suyo. Nada de pools de "finders" por dimensión
(correctness/seguridad/concurrencia/SQL/estilo) — eso sobra; el punto ya está
identificado, aquí solo se confirma o se cae.

### Ejemplo conceptual

> Puntos candidatos: (P1) `search.ts:88` no escapa el input antes del `LIKE`;
> (P2) `rerank.ts:20` usa un modelo proscrito; (P3) `es.ts` mete churn unicode.
>
> - Agente P1 → hace `git show`/`grep`, comprueba que el input llega crudo al
>   query → **confirmado 🚨** con `search.ts:88`.
> - Agente P2 → lee `rerank.ts`, ve que el modelo se sobreescribe por env más
>   abajo → **descartado** (otra capa lo tapa).
> - Agente P3 → diff de `es.ts`, confirma cambios unicode ajenos a la feature →
>   **matizado 🟢** (churn no relacionado que infla el diff).

### Comandos de verificación (contra la rama de la PR, no tu working tree)

```bash
B=origin/<rama-de-la-PR>
git show      $B:ruta/al/fichero.ts        # estado real del código
git grep -n   "patrón" $B -- 'glob'         # ¿existe / cuántas veces?
git cat-file -e $B:ruta/fichero && echo ok  # ¿el fichero existe?
git ls-tree -r --name-only $B | grep ...    # ficheros commiteados que no deberían
```

Descarta sin piedad lo que no aplique. Preferimos un falso negativo a publicar
un falso positivo.

---

## 4. Qué es un buen comentario (rubric de comentarios)

El listón: **pocos, cortos y load-bearing**. Un comentario existe para decir algo
que el código **no puede** expresar — un "por qué" no obvio, un invariante, una
trampa para el siguiente editor. Si borrarlo no pierde información que un lector
competente no pudiera recuperar del propio código, es ruido → bórralo. Cero
comentarios en un fichero es un estado sano. Antes de comentar código poco claro,
prefiere hacerlo auto-explicativo (mejor nombre, función extraída).

**Conservar:** el porqué (decisión sorprendente, regla de negocio), un aviso de
trampa no visible en el call site, un invariante que los tipos no codifican, un
enlace a causa externa (bug upstream/RFC), una rareza intencional marcada para que
nadie la "arregle", un TODO/FIXME con sustancia, o un doc comment de API pública
cuando la firma no basta.

**Borrar:** narrar la línea (`// incrementa el contador`), repetir el nombre
(`// Envía el email` sobre `sendEmail()`), cabeceras de sección en funciones cortas,
hablarle al reviewer (`// arreglado según feedback` → va al commit), describir el
diff/pasado (`// antes usaba X`), repetir tipos, código muerto comentado, relleno
(`// Helpers`), o explicar comportamiento estándar del lenguaje.

**De los que sobreviven:** una línea (dos máximo), en el idioma del código, sobre
su propia línea y adyacente a lo que explican. Cuando el código cambia, el
comentario es parte del diff: actualízalo o bórralo — uno obsoleto es peor que
ninguno.

**Al recortar:** no añadas comentarios nuevos (salvo una trampa peligrosa que
descubras, con un aviso de una línea) ni narración para "compensar" lo borrado. En
PRs de IA típicas se borra la mayoría y sobrevive un puñado.

---

## 5. Severidades

| Emoji | Nivel |
|-------|-------|
| 🔴 | Bloqueante (rompe, corrompe datos, agujero grave) |
| 🚨 | Seguridad (secretos, auth, input no validado) — siempre destacado |
| 🟡 | Menor / correctitud (bug real no crítico) |
| 🟢 | Estilo / nit (coherencia, churn, dead code, docstrings) |
| ✅ | Verificado correcto — **SOLO si se pide explícitamente** |

Por defecto **NO** hay sección de "lo que está bien / verificado correcto".
Solo se incluye si el receptor la pide.

---

## 6. Formato de salida

### 6a. Nomenclatura de hallazgos (depende del destino)

- **Review de GitHub (comentario formal):** numeración simple y secuencial
  cruzando severidades → `🔴 1`, `🚨 2`, `🟡 3`, `🟢 4`...
- **`.md` interno de trabajo:** códigos **severidad + índice** (`B3`, `A1`,
  `M5`, `S1`) para poder referenciarlos al aplicar fixes ("aplica B3 y B4").

### 6b. Estructura de cada hallazgo

```markdown
### 🔴 1. <Titular accionable en una línea>
`ruta/al/fichero.ts:línea`   (y `→ usado en :otra` si aplica)

<1-3 frases, tono llano: qué pasa y el impacto CONCRETO (coste/escala/contrato).>

**Fix:** <acción concreta; snippet solo si aclara>.
```

Reglas:
- **`path:línea` siempre**, clicable y verificable.
- Titular accionable ("X no valida Y", no "revisar X").
- Explicación breve; el valor está en el impacto concreto, no en la prosa.
- Ordena por severidad (🔴/🚨 primero).

### 6c. Sección Transversal

Al final de los hallazgos, gaps que no encajan en un punto concreto: timeouts,
rate-limiting, degradación ante fallos, accesibilidad, falta de tests, etc.

### 6d. Tabla resumen ejecutivo (en el `.md` interno)

Cierra el `.md` con:

| # | Severidad | Qué | Acción / Dónde |
|---|-----------|-----|----------------|
| B1 | 🔴 | ... | ... |
| M3 | 🟡 | ... | ... |

---

## 7. Verificación de hallazgos y de fixes

- **Antes de reportar un hallazgo:** verifícalo contra el árbol/DB de la rama
  (sección 2). Evita falsos positivos.
- **Al APLICAR fixes:** pruébalos **end-to-end de verdad** ("prueba que
  funcione"), no solo `tsc --noEmit`. Formas válidas, elige según el caso:
  - **Typecheck + mediciones reproducibles** (recall/rank, coste, latencia).
  - **Levantar el server / ejecutar el flujo real** — es una opción válida
    (todas las vars están en el `env`). Ofrécelo cuando aplique; no lo impongas.
  - Para cambios de modelo/prompt, ejecuta la llamada real y observa la salida.

---

## 8. Publicar la review (GitHub)

```bash
# Crear review pidiendo cambios:
gh api repos/<owner>/<repo>/pulls/<N>/reviews \
  -X POST -f event=REQUEST_CHANGES -F "body=@review.md" --jq '.html_url'

# Editar una review ya enviada (mismo id):
jq -Rs '{body: .}' review.md > body.json
gh api repos/<owner>/<repo>/pulls/<N>/reviews/<REVIEW_ID> \
  -X PUT --input body.json --jq '.html_url'
```

`event`: `REQUEST_CHANGES` (bloquea), `COMMENT` (sin bloquear), `APPROVE`.
Para comentarios inline por línea: mismo POST a `.../reviews` con array
`comments` (`path`, `line`, `body`).

> **NUNCA** commitees ni publiques una review formal sin **OK explícito** del
> usuario.

---

## 9. Flujo git de cierre

1. **Rama desde la FEATURE branch** (no desde `main`).
2. El `.md` consolidado se convierte en el **BODY de la PR** y **se borra de la
   rama** (no dejes el `.md` commiteado).
3. **Un solo commit**, subir, `gh pr create`.
4. Al final del body: **"Siguientes pasos"**, en lenguaje llano y breve.
5. Cierra de verdad lo pendiente y **verifícalo en el diff** (revisa los file
   changes: "no lo veo en file changes" = no está cerrado).

---

## 10. Orden de arreglo sugerido (para el receptor)

Prioriza: primero lo que **rompe o expone algo** (auth/🚨, coste, corrupción de
datos), luego bugs visibles al usuario, y al final limpieza/estilo. Respeta las
dependencias entre PRs (si #B se basa en #A, #A va antes).

---

## TL;DR del flujo

1. Base real de cada PR (no asumas `main`); revisa el **delta**.
2. Ten claro **qué cazar** (sección 1): coherencia con el código, impacto
   concreto, sobre-ingeniería, comentarios ruidosos, higiene/seguridad.
3. Arma la lista de **puntos** candidatos (commits + diff).
4. **Verifica cada punto adversarialmente** con grep/git/leer el fichero
   (paralelizable: un agente por punto). Confirma / descarta / matiza.
5. Audita los **comentarios** del diff con el rubric (sección 4): pocos, cortos,
   load-bearing; la mayoría de los generados por IA se borran.
6. Sección **Transversal** para lo que no encaja en un punto.
7. Escribe corto, en español llano: `path:línea` + impacto concreto + **Fix:**.
   Todas las severidades (🔴🚨🟡🟢). Sin sección de "lo correcto" salvo que se
   pida. Tabla resumen al final del `.md`.
8. Fixes probados **end-to-end**.
9. Cierre git: rama desde feature → `.md` como body de PR (borrado de la rama) →
   commit único → `gh pr` → "siguientes pasos". **Nada se publica sin OK.**
