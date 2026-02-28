# Diagrammes Mermaid Suggérés — Documentation Frontend

## 1. Architecture globale des Stores et Composables

**Type** : `graph TD` (DAG)

Montrerait le flux de dépendances : Composants → Composables → Stores → API Client → Backend. Illustre comment les données circulent depuis le backend jusqu'à l'affichage et les mutations locales du state. Utile pour comprendre l'architecture réactive globale en un coup d'oeil.

---

## 2. Flux d'authentification et guards

**Type** : `flowchart TD`

Cartographie le parcours utilisateur : login → checkAuth (session restore) → route guard (isAuthenticated?) → adminGuard (requiresAdmin?) → redirect ou access. Montre les redirection logiques et conditions. Utile pour onboarder rapidement sur la sécurité du routing.

---

## 3. Cycle de vie SSE et synchronisation temps réel

**Type** : `sequenceDiagram`

Acteurs : Frontend, Backend (SSE), Stores. Séquence : composant monte → useSSE() ouvre EventSource → backend envoie événement → EventSource dispatche → store met à jour → UI réagit. Auto-cleanup sur unmount. Illustre la responsabilité du realtime dans l'app.

---

## 4. État et transitions d'une exécution de run

**Type** : `stateDiagram-v2`

États : pending → running → (paused ↔ running) → completed/failed/cancelled. Actions associées : launch, pause, resume, cancel, retry. Montre les chemins valides et les dépendances entre les états (ex : retry seulement en failed). Utile pour éviter les bugs logiques du contrôle de run.

---

## 5. Anatomie d'une Feature (board comme exemple)

**Type** : `graph TD`

Structure : Features/board contient Composants (EpicCardGrid, StoryDetailPanel, etc.) → Composables (useStories, useRunLauncher) → Stores (epics, stories) → API (GET /epics, POST /runs). Zoom sur une feature typique pour illustrer le pattern isolation/responsabilité.

---

## 6. Pipeline de code génération (OpenAPI → Types)

**Type** : `flowchart LR`

Source : api/openapi.yaml → openapi-typescript (génère schema.d.ts) et oapi-codegen (backend). Frontend utilise les types générés pour openapi-fetch. Montre pourquoi la spec est source unique de vérité et les étapes de régénération. Critique pour éviter le skew entre spec et implémentation.

---

## 7. Hiérarchie des routes et lazy-loading

**Type** : `graph TD`

Racine : / (toutes les routes) → Routes authentifiées vs non-auth, puis par domaine (projects/, runs/, /admin). Marque celles lazy-loadées. Utile pour comprendre la stratégie de splitting du bundle et l'organisation logique du routeur.

---

## 8. Flux de validation et soumission de formulaire (exemplaire : formulaire story)

**Type** : `flowchart TD`

Utilisateur remplit form → vee-validate + zod valide → isValid? → submit API ou affiche erreurs → API repond (success/error) → toast notification ou mise à jour store. Montre l'intégration validation front ↔ backend et gestion des erreurs. Applicable à tous les formulaires.

---

## 9. Composition CSS (layers + Tailwind + PrimeVue)

**Type** : `graph BT` (bottom-to-top pour visualiser la cascade)

Base : tailwind-base (resets) ← primevue (composants) ← tailwind-utilities (overrides). Chaque layer a une zone de responsabilité. Aide à comprendre pourquoi les utilitaires surpassent PrimeVue et la stratégie de theming Aura.

---

## 10. Modèle de données SSE et événements temps réel

**Type** : `classDiagram`

Classe EventPayload : runId, stepId, eventType, data. Sous-classes : RunStartedEvent, StepCompletedEvent, LogEmittedEvent, HITLPendingEvent. Montre la hiérarchie et les champs critiques. Utile pour comprendre comment structurer les listeners et les handlers SSE.

