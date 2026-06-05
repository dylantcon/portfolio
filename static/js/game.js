import { API } from './api.js?v=8';

// FogOfWar handles visibility and exploration tracking
class FogOfWar {
    constructor() {
        this.explored = new Set();  // "x,y" -> tiles player has seen
        this.visible = new Set();   // Currently visible tiles
        this.visionRadius = 8;      // How far the player can see, in tiles
    }

    // Check if tile has been explored
    isExplored(x, y) {
        return this.explored.has(`${x},${y}`);
    }

    // Check if tile is currently visible
    isVisible(x, y) {
        return this.visible.has(`${x},${y}`);
    }

    // Calculate visibility using raycasting
    calculateVisibility(playerX, playerY, isOpaque) {
        this.visible.clear();

        // Always see the player's tile
        this.visible.add(`${playerX},${playerY}`);
        this.explored.add(`${playerX},${playerY}`);

        // Cast rays in all directions
        const numRays = 360;
        for (let i = 0; i < numRays; i++) {
            const angle = (i / numRays) * 2 * Math.PI;
            this.castRay(playerX, playerY, angle, isOpaque);
        }
    }

    castRay(startX, startY, angle, isOpaque) {
        const dx = Math.cos(angle);
        const dy = Math.sin(angle);

        let x = startX;
        let y = startY;

        // Rays step half a tile at a time, so a radius of N tiles takes 2N steps
        const steps = this.visionRadius * 2;
        for (let step = 0; step <= steps; step++) {
            const tileX = Math.round(x);
            const tileY = Math.round(y);
            const key = `${tileX},${tileY}`;

            // Mark as visible and explored
            this.visible.add(key);
            this.explored.add(key);

            // Stop ray if tile blocks vision
            if (isOpaque(tileX, tileY)) {
                break;
            }

            x += dx * 0.5;
            y += dy * 0.5;
        }
    }

    // Get visibility state for rendering
    getVisibilityState(x, y) {
        if (this.isVisible(x, y)) {
            return 'visible';
        } else if (this.isExplored(x, y)) {
            return 'explored';
        }
        return 'hidden';
    }
}

// ChunkManager handles loading, caching, and accessing chunk-based map data
class ChunkManager {
    constructor(api) {
        this.api = api;
        this.world = null;           // World manifest
        this.chunks = new Map();     // Loaded chunks: "x,y" -> chunk data
        this.loading = new Set();    // Chunks currently being fetched
        this.chunkSize = 50;         // Will be set from world manifest

        // Edge generation tiles
        this.waterTile = { char: '~', color: '#4da6ff' };
        this.sandTile = { char: '.', color: '#f4a460' };
        this.fogTile = { char: '`', color: '#333333' };
    }

    async init() {
        this.world = await this.api.getWorld();
        this.chunkSize = this.world.chunk_size;

        // Load spawn chunk immediately
        const [sx, sy] = this.world.spawn_chunk;
        await this.loadChunk(sx, sy);

        return {
            x: sx * this.chunkSize + this.world.spawn_local[0],
            y: sy * this.chunkSize + this.world.spawn_local[1]
        };
    }

    // Convert world coordinates to chunk coordinates
    worldToChunk(worldX, worldY) {
        return {
            chunkX: Math.floor(worldX / this.chunkSize),
            chunkY: Math.floor(worldY / this.chunkSize),
            localX: ((worldX % this.chunkSize) + this.chunkSize) % this.chunkSize,
            localY: ((worldY % this.chunkSize) + this.chunkSize) % this.chunkSize
        };
    }

    // Check if a chunk exists in the world
    chunkExists(chunkX, chunkY) {
        const key = `${chunkX},${chunkY}`;
        return this.world.available_chunks.hasOwnProperty(key);
    }

    // Load a chunk (returns promise)
    async loadChunk(chunkX, chunkY) {
        const key = `${chunkX},${chunkY}`;

        if (this.chunks.has(key) || this.loading.has(key)) {
            return; // Already loaded or loading
        }

        if (!this.chunkExists(chunkX, chunkY)) {
            return; // Chunk doesn't exist
        }

        this.loading.add(key);

        try {
            const chunk = await this.api.getChunk(chunkX, chunkY);
            if (chunk) {
                this.chunks.set(key, chunk);
            }
        } catch (e) {
            console.error(`Failed to load chunk ${key}:`, e);
        } finally {
            this.loading.delete(key);
        }
    }

    // Prefetch chunks around a position - when player's LOS can reach the border
    prefetchAround(worldX, worldY, visionRadius = 8) {
        const { chunkX, chunkY, localX, localY } = this.worldToChunk(worldX, worldY);
        // Load adjacent chunks when LOS can see into them
        const threshold = visionRadius;
        const maxLocal = this.chunkSize - 1 - threshold;

        // Always ensure current chunk is loaded
        this.loadChunk(chunkX, chunkY);

        // Check proximity to each edge and load adjacent chunks as needed
        const nearWest = localX <= threshold;
        const nearEast = localX >= maxLocal;
        const nearNorth = localY <= threshold;
        const nearSouth = localY >= maxLocal;

        // Cardinal directions
        if (nearWest) this.loadChunk(chunkX - 1, chunkY);
        if (nearEast) this.loadChunk(chunkX + 1, chunkY);
        if (nearNorth) this.loadChunk(chunkX, chunkY - 1);
        if (nearSouth) this.loadChunk(chunkX, chunkY + 1);

        // Diagonal chunks (only load when near a corner)
        if (nearNorth && nearWest) this.loadChunk(chunkX - 1, chunkY - 1);
        if (nearNorth && nearEast) this.loadChunk(chunkX + 1, chunkY - 1);
        if (nearSouth && nearWest) this.loadChunk(chunkX - 1, chunkY + 1);
        if (nearSouth && nearEast) this.loadChunk(chunkX + 1, chunkY + 1);
    }

    // Get tile at world coordinates
    getTile(worldX, worldY) {
        const { chunkX, chunkY, localX, localY } = this.worldToChunk(worldX, worldY);
        const key = `${chunkX},${chunkY}`;

        // Check if chunk exists in world
        if (!this.chunkExists(chunkX, chunkY)) {
            // Generate beach/water edge
            return this.getEdgeTile(worldX, worldY, chunkX, chunkY);
        }

        // Check if chunk is loaded
        if (!this.chunks.has(key)) {
            // Trigger load and show fog
            this.loadChunk(chunkX, chunkY);
            return this.fogTile;
        }

        // Get tile from loaded chunk
        const chunk = this.chunks.get(key);
        const char = chunk.tiles[localY]?.[localX];

        if (!char) {
            return this.fogTile;
        }

        // Look up tile definition
        const tileDef = this.world.tile_definitions[char];
        if (tileDef) {
            return { char: tileDef.char, color: tileDef.color };
        }

        return { char, color: '#808080' };
    }

    // Non-existent chunks are ocean - the designed chunks have their own coastlines
    getEdgeTile(worldX, worldY, chunkX, chunkY) {
        return this.waterTile;
    }

    // Check if position is walkable
    isWalkable(worldX, worldY) {
        const { chunkX, chunkY, localX, localY } = this.worldToChunk(worldX, worldY);
        const key = `${chunkX},${chunkY}`;

        // Can't walk into non-existent chunks (water/beach edge)
        if (!this.chunkExists(chunkX, chunkY)) {
            return false;
        }

        // Can't walk into unloaded chunks (wait for load)
        if (!this.chunks.has(key)) {
            return false;
        }

        const chunk = this.chunks.get(key);
        const char = chunk.tiles[localY]?.[localX];

        if (!char) return false;

        const tileDef = this.world.tile_definitions[char];
        return tileDef ? tileDef.walkable : true;
    }

    // Get zone at position
    getZoneAt(worldX, worldY) {
        const { chunkX, chunkY, localX, localY } = this.worldToChunk(worldX, worldY);
        const key = `${chunkX},${chunkY}`;

        if (!this.chunks.has(key)) return null;

        const chunk = this.chunks.get(key);
        for (const zone of chunk.zones || []) {
            if (localX >= zone.bounds.min_x && localX <= zone.bounds.max_x &&
                localY >= zone.bounds.min_y && localY <= zone.bounds.max_y) {
                return zone;
            }
        }
        return null;
    }

    // Get current tile type name
    getTileType(worldX, worldY) {
        const { chunkX, chunkY, localX, localY } = this.worldToChunk(worldX, worldY);
        const key = `${chunkX},${chunkY}`;

        if (!this.chunkExists(chunkX, chunkY)) {
            return 'water';
        }

        if (!this.chunks.has(key)) {
            return 'unknown';
        }

        const chunk = this.chunks.get(key);
        const char = chunk.tiles[localY]?.[localX];
        const tileDef = this.world.tile_definitions[char];
        return tileDef?.type || 'unknown';
    }

    // Check if a tile blocks vision
    isOpaque(worldX, worldY) {
        const { chunkX, chunkY, localX, localY } = this.worldToChunk(worldX, worldY);
        const key = `${chunkX},${chunkY}`;

        // Non-existent chunks don't block vision (water)
        if (!this.chunkExists(chunkX, chunkY)) {
            return false;
        }

        // Unloaded chunks block vision
        if (!this.chunks.has(key)) {
            return true;
        }

        const chunk = this.chunks.get(key);
        const char = chunk.tiles[localY]?.[localX];

        if (!char) return true;

        // Tiles that block vision (walls and dense trees)
        const opaqueTiles = new Set([
            '#',  // walls
            'W',  // wood walls
            'T',  // large trees
        ]);

        return opaqueTiles.has(char);
    }
}

// Main Game class
class Game {
    constructor() {
        this.api = new API();
        this.chunkManager = new ChunkManager(this.api);
        this.fogOfWar = new FogOfWar();
        this.viewport = document.getElementById('viewport');
        this.position = { x: 0, y: 0 };
        this.viewportWidth = 40;
        this.viewportHeight = 20;

        // ASCII display config. The grid fills the available width; the row count
        // is fixed so the box stays short (keeps the projects in view). The
        // user-controlled zoom scales the glyph size, so the box height and tile
        // size scale together. displayRows stays above the LOS diameter
        // (visionRadius * 2 + 1 = 17) so the sight circle always fits.
        this.displayRows = 39;
        this.minScale = 0.6;
        this.maxScale = 1.6;
        this.scaleStep = 0.1;
        this.displayScale = this.loadDisplayScale();

        // Key state for smooth movement
        this.keysDown = new Set();
        this.moveInterval = null;
        this.moveDelay = 120;

        // Fog of war colors
        this.hiddenColor = '#1a1a1a';
        this.exploredDim = 0.4;  // Brightness multiplier for explored tiles
    }

    async init() {
        try {
            // Size the display box up front so it reserves layout space and the
            // page doesn't shift while the first chunks are fetched.
            this.applyDisplayScale();
            this.calculateViewportSize();

            // Initialize chunk manager and get spawn position
            this.position = await this.chunkManager.init();

            // Prefetch surrounding chunks based on vision radius
            this.chunkManager.prefetchAround(this.position.x, this.position.y, this.fogOfWar.visionRadius);

            this.render();
            this.setupEventListeners();
            this.updateZoneInfo();
        } catch (error) {
            console.error('Failed to initialize game:', error);
            this.viewport.innerHTML = `<span style="color:#ff4444">Error: ${error.message}</span>`;
        }
    }

    // --- Display zoom -----------------------------------------------------

    loadDisplayScale() {
        try {
            const saved = parseFloat(localStorage.getItem('displayScale'));
            if (!isNaN(saved)) {
                return Math.min(this.maxScale, Math.max(this.minScale, saved));
            }
        } catch (e) { /* localStorage unavailable (private mode, etc.) */ }
        return 1;
    }

    applyDisplayScale() {
        document.documentElement.style.setProperty('--display-scale', this.displayScale);
        try {
            localStorage.setItem('displayScale', String(this.displayScale));
        } catch (e) { /* ignore persistence failures */ }
    }

    setDisplayScale(scale) {
        const clamped = Math.min(
            this.maxScale,
            Math.max(this.minScale, Math.round(scale * 10) / 10)
        );
        if (clamped === this.displayScale) return;
        this.displayScale = clamped;
        this.applyDisplayScale();
        this.calculateViewportSize();
        this.render();
    }

    calculateViewportSize() {
        const viewportRect = this.viewport.getBoundingClientRect();
        const style = window.getComputedStyle(this.viewport);

        const paddingX = parseFloat(style.paddingLeft) + parseFloat(style.paddingRight);
        const paddingY = parseFloat(style.paddingTop) + parseFloat(style.paddingBottom);
        const borderX = parseFloat(style.borderLeftWidth) + parseFloat(style.borderRightWidth);
        const borderY = parseFloat(style.borderTopWidth) + parseFloat(style.borderBottomWidth);

        // The box fills its track (CSS), so columns are derived from its own
        // rendered width. Only the height is set below to keep the box short.
        const availableWidth = viewportRect.width - paddingX - borderX;

        // Measure the font cell: horizontal advance per character and vertical
        // stride per row. Sampling many glyphs/lines averages out sub-pixel
        // rounding. The stride probe must be a block of real lines because a
        // single inline span reports its glyph box, not the line-box stride.
        const fontCSS = `font-family:${style.fontFamily};font-size:${style.fontSize};line-height:${style.lineHeight};white-space:pre;position:absolute;visibility:hidden;left:-9999px;top:0`;
        const sample = 50;

        const wProbe = document.createElement('span');
        wProbe.style.cssText = fontCSS;
        wProbe.textContent = 'X'.repeat(sample);
        document.body.appendChild(wProbe);
        const charWidth = wProbe.getBoundingClientRect().width / sample;
        document.body.removeChild(wProbe);

        const hProbe = document.createElement('div');
        hProbe.style.cssText = fontCSS;
        hProbe.textContent = Array(sample).fill('X').join('\n');
        document.body.appendChild(hProbe);
        const lineStride = hProbe.getBoundingClientRect().height / sample;
        document.body.removeChild(hProbe);

        // A monospace cell is taller than it is wide, so a circular line-of-sight
        // renders as a vertical ellipse. Squash glyphs vertically until the
        // rendered cell is square (height === width); the LOS then reads round.
        const squash = Math.max(0.1, Math.min(1, charWidth / lineStride));
        document.documentElement.style.setProperty('--glyph-squash-y', squash);

        // The squash is purely visual; the rendered row height (= charWidth once
        // cells are square) is what the box height is sized against.
        const rowHeight = lineStride * squash;

        // Columns fill the width; rows are fixed (min keeps the LOS circle whole
        // even on narrow screens, where overflow is clipped from the edges).
        let cols = Math.max(21, Math.floor(availableWidth / charWidth));
        let rows = this.displayRows;

        if (cols % 2 === 0) cols--;
        if (rows % 2 === 0) rows--;

        this.viewportWidth = cols;
        this.viewportHeight = rows;

        // Width fills the track (CSS); set only the height so the box stays short
        // and scales with the zoom, keeping the projects in view.
        this.viewport.style.height = `${Math.round(rows * rowHeight + paddingY + borderY)}px`;
    }

    setupEventListeners() {
        // Display zoom buttons
        const zoomOut = document.getElementById('zoom-out');
        const zoomReset = document.getElementById('zoom-reset');
        const zoomIn = document.getElementById('zoom-in');
        if (zoomOut) zoomOut.addEventListener('click', (e) => {
            this.setDisplayScale(this.displayScale - this.scaleStep);
            e.currentTarget.blur();
        });
        if (zoomReset) zoomReset.addEventListener('click', (e) => {
            this.setDisplayScale(1);
            e.currentTarget.blur();
        });
        if (zoomIn) zoomIn.addEventListener('click', (e) => {
            this.setDisplayScale(this.displayScale + this.scaleStep);
            e.currentTarget.blur();
        });

        document.addEventListener('keydown', (e) => {
            const key = e.key.toLowerCase();

            if (['w', 's', 'a', 'd', 'arrowup', 'arrowdown', 'arrowleft', 'arrowright'].includes(key)) {
                e.preventDefault();

                if (!this.keysDown.has(key)) {
                    this.keysDown.add(key);
                    this.handleMove(key);

                    if (!this.moveInterval) {
                        this.moveInterval = setInterval(() => this.processHeldKeys(), this.moveDelay);
                    }
                }
            } else if (key === 'e') {
                e.preventDefault();
                this.handleInspect();
            }
        });

        document.addEventListener('keyup', (e) => {
            const key = e.key.toLowerCase();
            this.keysDown.delete(key);

            if (this.keysDown.size === 0 && this.moveInterval) {
                clearInterval(this.moveInterval);
                this.moveInterval = null;
            }
        });

        let resizeTimeout;
        window.addEventListener('resize', () => {
            clearTimeout(resizeTimeout);
            resizeTimeout = setTimeout(() => {
                this.calculateViewportSize();
                this.render();
            }, 100);
        });

        window.addEventListener('blur', () => {
            this.keysDown.clear();
            if (this.moveInterval) {
                clearInterval(this.moveInterval);
                this.moveInterval = null;
            }
        });

        // Mobile controls
        this.setupMobileControls();
    }

    setupMobileControls() {
        const dpadButtons = document.querySelectorAll('.dpad-btn[data-dir]');
        const inspectBtn = document.getElementById('inspect-btn');

        // Direction mapping
        const dirMap = {
            'up': 'arrowup',
            'down': 'arrowdown',
            'left': 'arrowleft',
            'right': 'arrowright'
        };

        dpadButtons.forEach(btn => {
            const dir = btn.dataset.dir;
            const key = dirMap[dir];

            // Handle touch start - begin movement
            btn.addEventListener('touchstart', (e) => {
                e.preventDefault();
                this.handleMove(key);

                // Start repeat movement after delay
                this.touchMoveTimeout = setTimeout(() => {
                    this.touchMoveInterval = setInterval(() => {
                        this.handleMove(key);
                    }, this.moveDelay);
                }, 200);
            }, { passive: false });

            // Handle touch end - stop movement
            btn.addEventListener('touchend', (e) => {
                e.preventDefault();
                clearTimeout(this.touchMoveTimeout);
                clearInterval(this.touchMoveInterval);
            }, { passive: false });

            // Handle touch cancel
            btn.addEventListener('touchcancel', (e) => {
                clearTimeout(this.touchMoveTimeout);
                clearInterval(this.touchMoveInterval);
            });

            // Also support mouse clicks for testing
            btn.addEventListener('click', (e) => {
                e.preventDefault();
                this.handleMove(key);
            });
        });

        // Inspect button
        if (inspectBtn) {
            inspectBtn.addEventListener('touchstart', (e) => {
                e.preventDefault();
                this.handleInspect();
            }, { passive: false });

            inspectBtn.addEventListener('click', (e) => {
                e.preventDefault();
                this.handleInspect();
            });
        }
    }

    processHeldKeys() {
        for (const key of this.keysDown) {
            this.handleMove(key);
            break;
        }
    }

    handleMove(key) {
        let newX = this.position.x;
        let newY = this.position.y;

        switch (key) {
            case 'w': case 'arrowup': newY--; break;
            case 's': case 'arrowdown': newY++; break;
            case 'a': case 'arrowleft': newX--; break;
            case 'd': case 'arrowright': newX++; break;
            default: return;
        }

        if (this.chunkManager.isWalkable(newX, newY)) {
            this.position.x = newX;
            this.position.y = newY;

            // Prefetch chunks as player moves (based on vision radius)
            this.chunkManager.prefetchAround(newX, newY, this.fogOfWar.visionRadius);

            this.render();
            this.updateZoneInfo();
        }
    }

    render() {
        const halfW = Math.floor(this.viewportWidth / 2);
        const halfH = Math.floor(this.viewportHeight / 2);

        // Calculate visibility from player position
        this.fogOfWar.calculateVisibility(
            this.position.x,
            this.position.y,
            (x, y) => this.chunkManager.isOpaque(x, y)
        );

        const rows = [];

        for (let vy = 0; vy < this.viewportHeight; vy++) {
            let row = '';
            for (let vx = 0; vx < this.viewportWidth; vx++) {
                const mapX = this.position.x - halfW + vx;
                const mapY = this.position.y - halfH + vy;

                if (vx === halfW && vy === halfH) {
                    row += '<span style="color:#00ffff;font-weight:bold">$</span>';
                } else {
                    const visibility = this.fogOfWar.getVisibilityState(mapX, mapY);

                    if (visibility === 'hidden') {
                        // Unexplored - show fog
                        row += `<span style="color:${this.hiddenColor}">\`</span>`;
                    } else {
                        const tile = this.chunkManager.getTile(mapX, mapY);
                        if (visibility === 'explored') {
                            // Explored but not visible - dim the color
                            const dimColor = this.dimColor(tile.color, this.exploredDim);
                            row += `<span style="color:${dimColor}">${tile.char}</span>`;
                        } else {
                            // Fully visible
                            row += `<span style="color:${tile.color}">${tile.char}</span>`;
                        }
                    }
                }
            }
            rows.push(row);
        }

        this.viewport.innerHTML = `<div class="viewport-content">${rows.join('\n')}</div>`;
    }

    // Dim a hex color by a multiplier (0-1)
    dimColor(hexColor, multiplier) {
        // Handle shorthand and full hex colors
        let hex = hexColor.replace('#', '');
        if (hex.length === 3) {
            hex = hex[0] + hex[0] + hex[1] + hex[1] + hex[2] + hex[2];
        }

        const r = Math.floor(parseInt(hex.substr(0, 2), 16) * multiplier);
        const g = Math.floor(parseInt(hex.substr(2, 2), 16) * multiplier);
        const b = Math.floor(parseInt(hex.substr(4, 2), 16) * multiplier);

        return `#${r.toString(16).padStart(2, '0')}${g.toString(16).padStart(2, '0')}${b.toString(16).padStart(2, '0')}`;
    }

    updateZoneInfo() {
        const zoneInfoEl = document.getElementById('zone-info');
        const zone = this.chunkManager.getZoneAt(this.position.x, this.position.y);

        // Get current tile type and cardinal directions
        const currentType = this.chunkManager.getTileType(this.position.x, this.position.y);
        const northType = this.chunkManager.getTileType(this.position.x, this.position.y - 1);
        const southType = this.chunkManager.getTileType(this.position.x, this.position.y + 1);
        const eastType = this.chunkManager.getTileType(this.position.x + 1, this.position.y);
        const westType = this.chunkManager.getTileType(this.position.x - 1, this.position.y);

        const tileInfo = `<p class="tile-info">Standing on: ${currentType}</p>
            <p class="tile-directions">N:${northType} S:${southType} E:${eastType} W:${westType}</p>`;

        if (!zone) {
            zoneInfoEl.innerHTML = tileInfo + '<p class="hint">Explore the map to discover projects...</p>';
            document.getElementById('project-info').innerHTML = '';
            return;
        }

        zoneInfoEl.innerHTML = `
            ${tileInfo}
            <p class="zone-name">${zone.name}</p>
            <p class="zone-description">${zone.description}</p>
            ${zone.project_id ? '<p class="hint">Press E to inspect</p>' : ''}
        `;
    }

    async handleInspect() {
        const zone = this.chunkManager.getZoneAt(this.position.x, this.position.y);
        if (!zone || !zone.project_id) return;

        try {
            const project = await this.api.getProject(zone.project_id);
            this.showProjectInfo(project);
        } catch (error) {
            console.error('Failed to load project:', error);
        }
    }

    showProjectInfo(project) {
        const el = document.getElementById('project-info');
        const tech = project.tech_stack ? project.tech_stack.join(', ') : 'N/A';

        el.innerHTML = `
            <div style="margin-top:15px;padding-top:15px;border-top:1px solid #3a3a3a">
                <p class="project-title">${project.title}</p>
                <p>${project.description}</p>
                <p class="project-tech">Tech: ${tech}</p>
                <p class="project-tech">Year: ${project.year || 'N/A'}</p>
                <div class="project-links">
                    ${project.github_url ? `<a href="${project.github_url}" target="_blank">[GitHub]</a>` : ''}
                    ${project.live_url ? `<a href="${project.live_url}" target="_blank">[Live]</a>` : ''}
                </div>
            </div>
        `;

        // Scroll info panel to show project details on mobile
        requestAnimationFrame(() => {
            el.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
        });
    }
}

document.addEventListener('DOMContentLoaded', () => new Game().init());
