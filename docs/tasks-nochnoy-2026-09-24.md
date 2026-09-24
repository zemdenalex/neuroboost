# Ночной прогон 24.09 — задачи

> Денис спит, работа идёт без вопросов. Не мержить в `main`, прод не трогать.
> Очередь из его слов: бот → веб (далеко позади) → mini-app / Android (только разведка).

## D3 — бот
- [x] Final review fix pass: I3 (лишние чтения при аварии API), I4 (ℹ️ задач дня) — `e52dc18`
- [x] I1 — решение Дениса: паузу красить ⬛, плоские только дни до первого использования; строка в ℹ️
- [x] I2 — `calendar_cell` (цвет / полоска / оба) в ⚙️ → 📌 — `ed6746d`
- [ ] push `develop`, CI зелёный, `scripts/deploy-dev-bot.sh`
- [ ] чеклист `docs/proverka-bota-2026-09-24-zadachi-dnya-d3.md` (спека §11 + ширина клетки), открыть в Obsidian

## Веб — задачи дня
- [ ] разведка: что в вебе есть по задачам дня (ничего?), месячный вид, настройки
- [ ] спека `docs/superpowers/specs/2026-09-24-web-day-tasks-design.md`
- [ ] план `docs/superpowers/plans/…`
- [ ] TDD по плану; `pnpm typecheck && pnpm test --run && pnpm build`

## Если веб упёрся
- [ ] разведка Telegram mini-app / Android — спека без кода
