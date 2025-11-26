let myMark = null;
let gameState = null;
let ws = null;

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
            renderBoard();
            updateStatus();
            break;

        case 'error':
            console.error('Error:', msg.error);
            break;
    }
}

function renderBoard() {
    const boardEl = document.getElementById('board');
    boardEl.innerHTML = '';

    for (let y = 0; y < 3; y++) {
        for (let x = 0; x < 3; x++) {
            const cell = document.createElement('div');
            cell.className = 'cell';
            const value = gameState.board[y][x];

            if (value) {
                cell.textContent = value;
                cell.classList.add(value.toLowerCase());
            }

            cell.onclick = () => makeMove(x, y);
            boardEl.appendChild(cell);
        }
    }
}

function updateStatus() {
    const statusEl = document.getElementById('status');
    const resetBtn = document.getElementById('reset-btn');

    if (gameState.winner) {
        if (gameState.winner === 'draw') {
            statusEl.textContent = "It's a draw!";
        } else if (gameState.winner === myMark) {
            statusEl.textContent = "You win!";
        } else {
            statusEl.textContent = "You lose!";
        }
        resetBtn.style.display = 'inline-block';
    } else {
        if (gameState.turn === myMark) {
            statusEl.textContent = "Your turn!";
        } else {
            statusEl.textContent = "Waiting for opponent...";
        }
        resetBtn.style.display = 'none';
    }
}

function makeMove(x, y) {
    if (!gameState || gameState.winner) return;
    if (gameState.turn !== myMark) return;
    if (gameState.board[y][x] !== '') return;

    ws.send(JSON.stringify({ type: 'move', x: x, y: y }));
}

function resetGame() {
    ws.send(JSON.stringify({ type: 'reset' }));
}

// Set up reset button and start connection
document.getElementById('reset-btn').onclick = resetGame;
connect();
