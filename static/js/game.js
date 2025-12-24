import { API } from './api.js?v=7';

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
        this.fogTile = { char: '░', color: '#333333' };
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

    // Prefetch chunks around a position
    prefetchAround(worldX, worldY) {
        const { chunkX, chunkY } = this.worldToChunk(worldX, worldY);

        // Load 3x3 grid around current chunk
        for (let dy = -1; dy <= 1; dy++) {
            for (let dx = -1; dx <= 1; dx++) {
                this.loadChunk(chunkX + dx, chunkY + dy);
            }
        }
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
}

// Main Game class
class Game {
    constructor() {
        this.api = new API();
        this.chunkManager = new ChunkManager(this.api);
        this.viewport = document.getElementById('viewport');
        this.position = { x: 0, y: 0 };
        this.viewportWidth = 40;
        this.viewportHeight = 20;

        // Key state for smooth movement
        this.keysDown = new Set();
        this.moveInterval = null;
        this.moveDelay = 120;
    }

    async init() {
        try {
            // Initialize chunk manager and get spawn position
            this.position = await this.chunkManager.init();

            // Prefetch surrounding chunks
            this.chunkManager.prefetchAround(this.position.x, this.position.y);

            this.calculateViewportSize();
            this.render();
            this.setupEventListeners();
            this.updateZoneInfo();
        } catch (error) {
            console.error('Failed to initialize game:', error);
            this.viewport.innerHTML = `<span style="color:#ff4444">Error: ${error.message}</span>`;
        }
    }

    calculateViewportSize() {
        const viewportRect = this.viewport.getBoundingClientRect();
        const style = window.getComputedStyle(this.viewport);

        const paddingX = parseFloat(style.paddingLeft) + parseFloat(style.paddingRight);
        const paddingY = parseFloat(style.paddingTop) + parseFloat(style.paddingBottom);
        const borderX = parseFloat(style.borderLeftWidth) + parseFloat(style.borderRightWidth);
        const borderY = parseFloat(style.borderTopWidth) + parseFloat(style.borderBottomWidth);

        const availableWidth = viewportRect.width - paddingX - borderX;
        const availableHeight = viewportRect.height - paddingY - borderY;

        const testSpan = document.createElement('span');
        testSpan.style.cssText = `font-family:${style.fontFamily};font-size:${style.fontSize};line-height:${style.lineHeight};position:absolute;visibility:hidden;white-space:pre`;
        testSpan.textContent = 'X';
        document.body.appendChild(testSpan);
        const charWidth = testSpan.getBoundingClientRect().width;
        const charHeight = testSpan.getBoundingClientRect().height;
        document.body.removeChild(testSpan);

        let cols = Math.max(20, Math.min(Math.floor(availableWidth / charWidth), 100));
        let rows = Math.max(10, Math.min(Math.floor(availableHeight / charHeight), 50));

        if (cols % 2 === 0) cols--;
        if (rows % 2 === 0) rows--;

        this.viewportWidth = cols;
        this.viewportHeight = rows;
    }

    setupEventListeners() {
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

            // Prefetch chunks as player moves
            this.chunkManager.prefetchAround(newX, newY);

            this.render();
            this.updateZoneInfo();
        }
    }

    render() {
        const halfW = Math.floor(this.viewportWidth / 2);
        const halfH = Math.floor(this.viewportHeight / 2);

        const rows = [];

        for (let vy = 0; vy < this.viewportHeight; vy++) {
            let row = '';
            for (let vx = 0; vx < this.viewportWidth; vx++) {
                const mapX = this.position.x - halfW + vx;
                const mapY = this.position.y - halfH + vy;

                if (vx === halfW && vy === halfH) {
                    row += '<span style="color:#00ffff;font-weight:bold">$</span>';
                } else {
                    const tile = this.chunkManager.getTile(mapX, mapY);
                    row += `<span style="color:${tile.color}">${tile.char}</span>`;
                }
            }
            rows.push(row);
        }

        this.viewport.innerHTML = `<div class="viewport-content">${rows.join('\n')}</div>`;
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
    }
}

document.addEventListener('DOMContentLoaded', () => new Game().init());
