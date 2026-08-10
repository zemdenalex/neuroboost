---
id: blocker-ssh-key-mismatch-deployment
title: "Снят: 193.104.57.79 — вообще не тот сервер. NeuroBoost живёт на 62.76.228.106"
type: learning
status: verified
verified_by: session-88767014
verified_at: 2026-08-10
tags: [neuroboost, deploy, ssh, security, resolved]
weight: { importance: 3, connectivity: 3, access: 4, last_accessed: 2026-08-10 }
sources:
  - file: ".remember/night-loop-2026-08-10.md §4"
  - command: "ssh -o BatchMode=yes root@62.76.228.106 'docker ps'"
stakes: high
links:
  - relates-to: workitem-p2-notifications-last-mile
  - relates-to: entity-server-topology
---
**Блокера нет.** Узел был заведён консолидатором 10.08 ~03:25 по промежуточному состоянию
и утверждал, что деплой упирается в несовпадение host-key у `193.104.57.79`. Это оказалось
**ложным следом**: по этому адресу NeuroBoost никогда и не жил.

**Что выяснено через полчаса, живьём:**

- Настоящий сервер — **`root@62.76.228.106`**, ключ-авторизация уже работает
  (`ssh -o BatchMode=yes` проходит без пароля).
- На нём **обе** среды: `neuroboost-api` / `-db` / `-web` / `-bot` (prod, `/opt/neuroboost`)
  и `neuroboost-dev-api` / `-dev-db` (staging, `/root/neuroboost-dev`).
- `193.104.57.79` — другая машина. Денис предположил, что это бот-хост. **Не выяснено, и
  выяснять незачем**: писать туда ничего не нужно.

**Что из старого узла остаётся верным** — протокол на случай, когда host-key всё же
разошёлся: `ssh-keygen -R`, затем `accept-new`, затем **проверить внутренность**
(`ls -d /opt/neuroboost`, `docker ps`) и только потом писать секрет. Это правило общее и
переживает конкретный адрес.

**Урок для графа, а не для деплоя:** узел, рождённый из промежуточного состояния сессии,
называет блокером то, что через полчаса работы оказалось опечаткой в поиске. Такие узлы
обязаны иметь `sources.command` — воспроизводимую проверку, а не ссылку на строки
черновика. Ср. [[learning-checkbox-in-a-plan-is-a-claim-not-evidence]]: заявка — не
свидетельство, и в графе это работает так же, как в плане.
