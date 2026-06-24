---
# Configuration La Manufacture (per-repo, VERSIONNEE). Partagee par forge + atelier.
# Generee par /atelier:init. Ne contient QUE les overrides que la detection auto ne couvre pas.
# JAMAIS de secret : utiliser ${ENV_VAR} ou le Keychain.

# --- Tracker : GitHub Projects v2 (#1 "hopeitworks Board") ---------------
# detect-context emet "github-issues" par defaut sur un remote github ; on force
# github-projects car le backlog est pilote dans le Project v2 lie au repo.
tracker:
  platform: github-projects
  ticket_regex: '^#?[0-9]+$'
  github_project_owner: zkarahacane
  github_project_number: 1
  status_field: Status
  story_points_field: Story Points
  # Colonnes du board : Backlog -> Specified -> Architected -> In Progress -> Review -> Testing -> Done
  transitions:
    dev_start: ["In Progress"]
    in_review: ["Review"]
    delivered: ["Done"]
  # Etats "prets a developper" (forge:status / autopilot piochent ici).
  # Le board n'a pas de colonne "Ready"/"Todo". "Specified" = sortie atelier (story+AC+SP) prete pour l'ingenierie.
  # On inclut "Backlog" pour que /forge:status et /forge:develop voient les 18 stories actuelles comme developpables.
  ready_states: ["Backlog", "Specified", "Architected"]

# --- VCS ------------------------------------------------------------------
vcs:
  merge_method: squash

# --- Stacks (ports QA locale, alignes sur le stack docker du projet) ------
stacks:
  default_ports:
    front: 5173
    back: 8080

# --- Atelier (amont produit : BA/PO) -------------------------------------
# socle_root non renseigne : resolu par la sonde ~/.claude/plugins/socle (symlink).
atelier:
  default_prioritization: rice
  max_story_points: 8
---

# Configuration La Manufacture - hopeitworks

Genere par `/atelier:init`. Tracker = **GitHub Project #1** ("hopeitworks Board",
https://github.com/users/zkarahacane/projects/1). Adapter les sections ci-dessus puis
relancer `/atelier:init` pour revalider le profil de contexte.
