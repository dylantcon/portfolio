export class API {
    constructor() {
        this.baseURL = '/api';
    }

    // World manifest - contains tile definitions, spawn info, available chunks
    async getWorld() {
        const response = await fetch(`${this.baseURL}/world`);
        if (!response.ok) {
            throw new Error('Failed to fetch world');
        }
        return await response.json();
    }

    // Individual chunk by grid coordinates
    async getChunk(x, y) {
        const response = await fetch(`${this.baseURL}/chunks/${x}/${y}`);
        if (!response.ok) {
            if (response.status === 404) {
                return null; // Chunk doesn't exist
            }
            throw new Error(`Failed to fetch chunk ${x},${y}`);
        }
        return await response.json();
    }

    // Legacy: full map (kept for compatibility)
    async getFullMap() {
        const response = await fetch(`${this.baseURL}/game/map`);
        if (!response.ok) {
            throw new Error('Failed to fetch map');
        }
        return await response.json();
    }

    async getProjects() {
        const response = await fetch(`${this.baseURL}/projects`);
        if (!response.ok) {
            throw new Error('Failed to fetch projects');
        }
        return await response.json();
    }

    async getProject(id) {
        const response = await fetch(`${this.baseURL}/projects/${id}`);
        if (!response.ok) {
            throw new Error('Failed to fetch project');
        }
        return await response.json();
    }
}
