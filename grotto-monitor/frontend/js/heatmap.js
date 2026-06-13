class HeatmapRenderer {
    constructor() {
        this.points = [];
    }

    setData(data) {
        this.points = data || [];
    }

    addPoint(x, y, intensity) {
        this.points.push({ x, y, intensity });
    }

    render(canvas, data) {
        if (data) this.setData(data);
        
        const ctx = canvas.getContext('2d');
        const width = canvas.width;
        const height = canvas.height;

        ctx.clearRect(0, 0, width, height);

        if (this.points.length === 0) {
            this.renderDefault(ctx, width, height);
            return;
        }

        this.points.forEach(point => {
            const x = (point.x / 40 + 0.5) * width;
            const y = (point.y / 40 + 0.5) * height;
            const intensity = point.intensity || 0.5;
            const radius = 30 + intensity * 50;

            const gradient = ctx.createRadialGradient(x, y, 0, x, y, radius);
            const alpha = intensity * 0.8;
            gradient.addColorStop(0, `rgba(255, 0, 0, ${alpha})`);
            gradient.addColorStop(0.3, `rgba(255, 150, 0, ${alpha * 0.7})`);
            gradient.addColorStop(0.6, `rgba(255, 255, 0, ${alpha * 0.4})`);
            gradient.addColorStop(1, 'rgba(0, 255, 0, 0)');

            ctx.fillStyle = gradient;
            ctx.beginPath();
            ctx.arc(x, y, radius, 0, Math.PI * 2);
            ctx.fill();
        });

        this.applyColorMapping(ctx, canvas);
    }

    renderDefault(ctx, width, height) {
        const centerX = width / 2;
        const centerY = height / 2;

        for (let i = 0; i < 8; i++) {
            const angle = (i / 8) * Math.PI * 2;
            const distance = 80 + Math.random() * 100;
            const x = centerX + Math.cos(angle) * distance;
            const y = centerY + Math.sin(angle) * distance;
            const intensity = 0.3 + Math.random() * 0.5;
            const radius = 40 + intensity * 60;

            const gradient = ctx.createRadialGradient(x, y, 0, x, y, radius);
            const alpha = intensity * 0.7;
            gradient.addColorStop(0, `rgba(255, ${Math.floor(100 + intensity * 100)}, 0, ${alpha})`);
            gradient.addColorStop(0.4, `rgba(255, 255, 0, ${alpha * 0.5})`);
            gradient.addColorStop(1, 'rgba(0, 255, 0, 0)');

            ctx.fillStyle = gradient;
            ctx.beginPath();
            ctx.arc(x, y, radius, 0, Math.PI * 2);
            ctx.fill();
        }

        this.applyColorMapping(ctx, { width, height });
    }

    applyColorMapping(ctx, canvas) {
        const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
        const data = imageData.data;

        for (let i = 0; i < data.length; i += 4) {
            const r = data[i];
            const g = data[i + 1];
            const b = data[i + 2];
            const a = data[i + 3];

            if (a > 0) {
                const intensity = (r + g + b) / (3 * 255);
                
                let newR, newG, newB;
                if (intensity < 0.25) {
                    newR = 0;
                    newG = Math.floor(intensity * 4 * 255);
                    newB = 255;
                } else if (intensity < 0.5) {
                    newR = 0;
                    newG = 255;
                    newB = Math.floor((0.5 - intensity) * 4 * 255);
                } else if (intensity < 0.75) {
                    newR = Math.floor((intensity - 0.5) * 4 * 255);
                    newG = 255;
                    newB = 0;
                } else {
                    newR = 255;
                    newG = Math.floor((1 - intensity) * 4 * 255);
                    newB = 0;
                }

                data[i] = newR;
                data[i + 1] = newG;
                data[i + 2] = newB;
            }
        }

        ctx.putImageData(imageData, 0, 0);
    }

    renderToCanvas(canvasId, data) {
        const canvas = document.getElementById(canvasId);
        if (canvas) {
            this.render(canvas, data);
        }
    }
}

window.heatmapRenderer = new HeatmapRenderer();
