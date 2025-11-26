let myMark = null;
let gameState = null;
let ws = null;
let selectedCell = null; // {x, y} of selected unit
let pendingGameState = null; // Game state to apply after combat animation
let combatState = null; // Tracks current combat {attackerMark, defenderMark, attackerRolled, defenderRolled, myRoll}

const BOARD_SIZE = 9;
const DICE_FACES = ['⚀', '⚁', '⚂', '⚃', '⚄', '⚅']; // 1-6

function connect() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

    ws.onopen = function() {
        console.log('Connected');
    };

    ws.onclose = function() {
        document.getElementById('status').textContent = 'Disconnected - refresh to reconnect';
    };

    ws.onmessage = function(event) {
        const msg = JSON.parse(event.data);
        console.log('Received:', msg);
        handleMessage(msg);
    };
}

function handleMessage(msg) {
    switch (msg.type) {
        case 'assigned':
            myMark = msg.mark;
            document.getElementById('player-info').textContent = `You are: ${myMark}`;
            break;

        case 'state':
            gameState = msg.game;
            selectedCell = null;
            renderBoard();
            updateStatus();
            break;

        case 'combat_start':
            // Combat initiated - show overlay with clickable dice
            showCombatStart(msg.combat);
            break;

        case 'combat_rolled':
            // A player clicked their dice
            handleCombatRolled(msg.combat);
            break;

        case 'combat':
            // Both rolled - show final result
            pendingGameState = msg.game;
            showCombatResult(msg.combat);
            break;

        case 'error':
            console.error('Error:', msg.error);
            break;

        case 'chat':
            addChatMessage(msg.from, msg.name, msg.message);
            break;
    }
}

// Dice animation functions
function getDiceFace(value) {
    return DICE_FACES[value - 1] || '⚀';
}

// Show combat overlay when combat starts - dice are clickable
function showCombatStart(combat) {
    const overlay = document.getElementById('combat-overlay');
    const attackerLabel = document.getElementById('attacker-label');
    const defenderLabel = document.getElementById('defender-label');
    const attackerDice = document.getElementById('attacker-dice');
    const defenderDice = document.getElementById('defender-dice');
    const attackerResult = document.getElementById('attacker-result');
    const defenderResult = document.getElementById('defender-result');
    const attackerDamage = document.getElementById('attacker-damage');
    const defenderDamage = document.getElementById('defender-damage');

    // Store combat state
    combatState = {
        attackerMark: combat.attackerMark,
        defenderMark: combat.defenderMark,
        attackerRolled: false,
        defenderRolled: false,
        attackerInterval: null,
        defenderInterval: null
    };

    // Set up labels
    attackerLabel.textContent = combat.attackerMark + ' (attacker)';
    attackerLabel.className = 'combatant-label ' + combat.attackerMark.toLowerCase();
    defenderLabel.textContent = combat.defenderMark + ' (defender)';
    defenderLabel.className = 'combatant-label ' + combat.defenderMark.toLowerCase();

    // Reset state
    attackerDice.textContent = '⚀';
    defenderDice.textContent = '⚀';
    attackerResult.textContent = '';
    attackerResult.className = 'combat-result';
    defenderResult.textContent = '';
    defenderResult.className = 'combat-result';
    attackerDamage.textContent = '';
    attackerDamage.className = 'damage-number';
    defenderDamage.textContent = '';
    defenderDamage.className = 'damage-number';

    // Attacker always rolls first
    const isAttacker = myMark === combat.attackerMark;
    const isDefender = myMark === combat.defenderMark;

    attackerDice.classList.remove('clickable', 'rolling', 'waiting');
    defenderDice.classList.remove('clickable', 'rolling', 'waiting');

    // Attacker's dice is clickable if you're the attacker
    if (isAttacker) {
        attackerDice.classList.add('clickable');
        attackerResult.textContent = 'Click to roll!';
    } else {
        attackerDice.classList.add('waiting');
        attackerResult.textContent = 'Waiting...';
    }

    // Defender must wait for attacker to roll first
    defenderDice.classList.add('waiting');
    defenderResult.textContent = 'Waiting for attacker...';

    // Show overlay
    overlay.classList.add('active');
}

// Handle dice click - send roll to server
function handleDiceClick(side) {
    if (!combatState) return;

    const isAttacker = myMark === combatState.attackerMark;
    const isDefender = myMark === combatState.defenderMark;

    // Only allow clicking your own dice
    if (side === 'attacker' && !isAttacker) return;
    if (side === 'defender' && !isDefender) return;

    // Don't allow double-clicking
    if (side === 'attacker' && combatState.attackerRolled) return;
    if (side === 'defender' && combatState.defenderRolled) return;

    // Send roll to server
    ws.send(JSON.stringify({ type: 'roll' }));
}

// Handle when server confirms a player rolled
function handleCombatRolled(combat) {
    if (!combatState) return;

    const attackerDice = document.getElementById('attacker-dice');
    const defenderDice = document.getElementById('defender-dice');
    const attackerResult = document.getElementById('attacker-result');
    const defenderResult = document.getElementById('defender-result');

    const isDefender = myMark === combatState.defenderMark;

    // Check if attacker just rolled
    if (combat.attackerRolled && !combatState.attackerRolled) {
        combatState.attackerRolled = true;
        combatState.attackerRollValue = combat.attackerRoll; // Store the roll value
        attackerDice.classList.remove('clickable', 'waiting');
        attackerDice.classList.add('rolling');
        attackerResult.textContent = 'Rolling...';

        // Start rolling animation
        combatState.attackerInterval = setInterval(() => {
            attackerDice.textContent = getDiceFace(Math.floor(Math.random() * 6) + 1);
        }, 100);

        // After 1 second, stop and show the result
        setTimeout(() => {
            if (combatState && combatState.attackerInterval) {
                clearInterval(combatState.attackerInterval);
                combatState.attackerInterval = null;
            }
            attackerDice.classList.remove('rolling');
            attackerDice.textContent = getDiceFace(combat.attackerRoll);
            attackerResult.textContent = `Rolled ${combat.attackerRoll}!`;

            // Now defender can roll!
            defenderDice.classList.remove('waiting');
            if (isDefender) {
                defenderDice.classList.add('clickable');
                defenderResult.textContent = `Beat ${combat.attackerRoll} to win! Click to roll!`;
            } else {
                defenderDice.classList.add('waiting');
                defenderResult.textContent = 'Waiting...';
            }
        }, 1000);
    }

    // Check if defender just rolled
    if (combat.defenderRolled && !combatState.defenderRolled) {
        combatState.defenderRolled = true;
        defenderDice.classList.remove('clickable', 'waiting');
        defenderDice.classList.add('rolling');
        defenderResult.textContent = 'Rolling...';

        // Start rolling animation
        combatState.defenderInterval = setInterval(() => {
            defenderDice.textContent = getDiceFace(Math.floor(Math.random() * 6) + 1);
        }, 100);
    }
}

// Show final combat result after both rolled
function showCombatResult(combat) {
    const attackerDice = document.getElementById('attacker-dice');
    const defenderDice = document.getElementById('defender-dice');
    const attackerResult = document.getElementById('attacker-result');
    const defenderResult = document.getElementById('defender-result');
    const attackerDamage = document.getElementById('attacker-damage');
    const defenderDamage = document.getElementById('defender-damage');

    // Clear any rolling intervals
    if (combatState) {
        if (combatState.attackerInterval) clearInterval(combatState.attackerInterval);
        if (combatState.defenderInterval) clearInterval(combatState.defenderInterval);
    }

    // Stop rolling animations
    attackerDice.classList.remove('rolling', 'clickable', 'waiting');
    defenderDice.classList.remove('rolling', 'clickable', 'waiting');

    // Show final dice values
    attackerDice.textContent = getDiceFace(combat.attackerRoll);
    defenderDice.textContent = getDiceFace(combat.defenderRoll);

    // Show roll values
    attackerResult.textContent = `Rolled ${combat.attackerRoll}`;
    defenderResult.textContent = `Rolled ${combat.defenderRoll}`;

    // After a beat, show winner and damage
    setTimeout(() => {
        if (combat.winner === 'attacker') {
            attackerResult.textContent = `Rolled ${combat.attackerRoll} - WINS!`;
            attackerResult.className = 'combat-result hit';
            defenderResult.textContent = `Rolled ${combat.defenderRoll} - loses`;
            defenderResult.className = 'combat-result miss';

            setTimeout(() => {
                defenderDamage.textContent = `-${combat.damage}`;
                defenderDamage.className = 'damage-number show';
            }, 300);
        } else {
            defenderResult.textContent = `Rolled ${combat.defenderRoll} - WINS!`;
            defenderResult.className = 'combat-result hit';
            attackerResult.textContent = `Rolled ${combat.attackerRoll} - loses`;
            attackerResult.className = 'combat-result miss';

            setTimeout(() => {
                attackerDamage.textContent = `-${combat.damage}`;
                attackerDamage.className = 'damage-number show';
            }, 300);
        }

        // After showing results, hide overlay
        setTimeout(() => {
            hideCombatOverlay();
        }, 2500);
    }, 800);
}

function hideCombatOverlay() {
    const overlay = document.getElementById('combat-overlay');
    overlay.classList.remove('active');

    // Clear combat state
    combatState = null;

    // Apply the pending game state
    if (pendingGameState) {
        gameState = pendingGameState;
        pendingGameState = null;
        selectedCell = null;
        renderBoard();
        updateStatus();
    }
}

function addChatMessage(from, name, message) {
    const chatMessages = document.getElementById('chat-messages');
    const msgEl = document.createElement('div');
    msgEl.className = 'chat-message';

    let displayName = from;
    if (name) {
        displayName = `${name} (${from})`;
    }

    msgEl.innerHTML = `<span class="from-${from.toLowerCase()}">${escapeHtml(displayName)}:</span> ${escapeHtml(message)}`;
    chatMessages.appendChild(msgEl);
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function sendChat() {
    const input = document.getElementById('chat-input');
    const message = input.value.trim();
    if (message) {
        ws.send(JSON.stringify({ type: 'chat', message: message }));
        input.value = '';
    }
}

// Get the current player's unit
function getMyUnit() {
    if (myMark === 'X') return gameState.unitX;
    if (myMark === 'O') return gameState.unitO;
    return null;
}

// Get the enemy's unit
function getEnemyUnit() {
    if (myMark === 'X') return gameState.unitO;
    if (myMark === 'O') return gameState.unitX;
    return null;
}

// Calculate Chebyshev distance (max of dx, dy - allows diagonal movement)
function getDistance(x1, y1, x2, y2) {
    const dx = Math.abs(x1 - x2);
    const dy = Math.abs(y1 - y2);
    return Math.max(dx, dy);
}

// Check if within movement range (3 squares)
function isWithinMoveRange(x1, y1, x2, y2) {
    const dist = getDistance(x1, y1, x2, y2);
    return dist > 0 && dist <= 3;
}

// Check if within attack range (1 square)
function isWithinAttackRange(x1, y1, x2, y2) {
    const dist = getDistance(x1, y1, x2, y2);
    return dist === 1;
}

// Check if a move to (x, y) is valid for the current player
function isValidMove(x, y) {
    if (!gameState || gameState.winner) return false;
    if (gameState.turn !== myMark) return false;

    const unit = getMyUnit();
    if (!unit) return false;

    // Must be within move range (3 squares)
    if (!isWithinMoveRange(unit.x, unit.y, x, y)) return false;

    // Must be empty
    if (gameState.board[y][x] !== '') return false;

    // Must be in bounds
    if (x < 0 || x >= BOARD_SIZE || y < 0 || y >= BOARD_SIZE) return false;

    return true;
}

// Check if attacking at (x, y) is valid
function isValidAttack(x, y) {
    if (!gameState || gameState.winner) return false;
    if (gameState.turn !== myMark) return false;

    const myUnit = getMyUnit();
    const enemyUnit = getEnemyUnit();
    if (!myUnit || !enemyUnit) return false;

    // Target must be the enemy position
    if (enemyUnit.x !== x || enemyUnit.y !== y) return false;

    // Enemy must be within attack range (1 square)
    if (!isWithinAttackRange(myUnit.x, myUnit.y, x, y)) return false;

    return true;
}

function renderBoard() {
    const boardEl = document.getElementById('board');
    boardEl.innerHTML = '';

    const myUnit = getMyUnit();

    for (let y = 0; y < BOARD_SIZE; y++) {
        for (let x = 0; x < BOARD_SIZE; x++) {
            const cell = document.createElement('div');
            cell.className = 'cell';
            cell.dataset.x = x;
            cell.dataset.y = y;

            const value = gameState.board[y][x];

            if (value) {
                cell.textContent = value;
                cell.classList.add(value.toLowerCase());

                // Add HP bar for units
                const unit = value === 'X' ? gameState.unitX : gameState.unitO;
                if (unit && unit.hp > 0) {
                    const hpBar = document.createElement('div');
                    hpBar.className = 'hp-bar';
                    const hpFill = document.createElement('div');
                    hpFill.className = 'hp-fill';
                    const hpPercent = (unit.hp / unit.maxHp) * 100;
                    if (hpPercent <= 30) {
                        hpFill.classList.add('low');
                    }
                    hpFill.style.width = `${hpPercent}%`;
                    hpBar.appendChild(hpFill);
                    cell.appendChild(hpBar);
                }
            }

            // Highlight selected cell
            if (selectedCell && selectedCell.x === x && selectedCell.y === y) {
                cell.classList.add('selected');
            }

            // Highlight valid moves/attacks when a unit is selected
            if (selectedCell && myUnit) {
                if (isValidMove(x, y)) {
                    cell.classList.add('valid-move');
                } else if (isValidAttack(x, y)) {
                    cell.classList.add('valid-attack');
                }
            }

            cell.onclick = () => handleCellClick(x, y);
            boardEl.appendChild(cell);
        }
    }
}

function handleCellClick(x, y) {
    if (!gameState || gameState.winner) return;
    if (gameState.turn !== myMark) return;

    const myUnit = getMyUnit();
    if (!myUnit) return;

    // If clicking on my own unit - select it
    if (myUnit.x === x && myUnit.y === y) {
        if (selectedCell && selectedCell.x === x && selectedCell.y === y) {
            // Clicking selected unit again - deselect
            selectedCell = null;
        } else {
            // Select this unit
            selectedCell = { x, y };
        }
        renderBoard();
        return;
    }

    // If no unit selected, do nothing
    if (!selectedCell) return;

    // If clicking on valid move target - move
    if (isValidMove(x, y)) {
        ws.send(JSON.stringify({ type: 'move', x: x, y: y }));
        return;
    }

    // If clicking on valid attack target - attack
    if (isValidAttack(x, y)) {
        ws.send(JSON.stringify({ type: 'attack', x: x, y: y }));
        return;
    }

    // Clicking elsewhere - deselect
    selectedCell = null;
    renderBoard();
}

function updateStatus() {
    const statusEl = document.getElementById('status');
    const resetBtn = document.getElementById('reset-btn');

    // Build HP info with max
    const xHP = gameState.unitX ? gameState.unitX.hp : 0;
    const xMaxHP = gameState.unitX ? gameState.unitX.maxHp : 10;
    const oHP = gameState.unitO ? gameState.unitO.hp : 0;
    const oMaxHP = gameState.unitO ? gameState.unitO.maxHp : 10;
    const hpInfo = `X: ${xHP}/${xMaxHP} HP | O: ${oHP}/${oMaxHP} HP`;

    if (gameState.winner) {
        if (gameState.winner === myMark) {
            statusEl.textContent = `You win! (${hpInfo})`;
        } else {
            statusEl.textContent = `You lose! (${hpInfo})`;
        }
        resetBtn.style.display = 'inline-block';
    } else {
        if (gameState.turn === myMark) {
            statusEl.textContent = `Your turn! ${hpInfo}`;
        } else {
            statusEl.textContent = `Waiting... ${hpInfo}`;
        }
        resetBtn.style.display = 'none';
    }
}

function resetGame() {
    ws.send(JSON.stringify({ type: 'reset' }));
}

function setName() {
    const input = document.getElementById('name-input');
    const name = input.value.trim();
    if (name) {
        ws.send(JSON.stringify({ type: 'setName', name: name }));
    }
}

// Set up event listeners and start connection
document.getElementById('reset-btn').onclick = resetGame;
document.getElementById('chat-send').onclick = sendChat;
document.getElementById('chat-input').addEventListener('keypress', function(e) {
    if (e.key === 'Enter') sendChat();
});
document.getElementById('name-btn').onclick = setName;
document.getElementById('name-input').addEventListener('keypress', function(e) {
    if (e.key === 'Enter') setName();
});
connect();
