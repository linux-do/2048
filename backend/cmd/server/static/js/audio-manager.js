class AudioManager {
    constructor() {
        this.audioContext = null;
        this.sounds = {};
        this.enabled = true;
        this.volume = 0.5;
        
        this.initAudioContext();
        this.createSounds();
    }

    initAudioContext() {
        try {
            // 创建音频上下文
            this.audioContext = new (window.AudioContext || window.webkitAudioContext)();
        } catch (e) {
            console.warn('Web Audio API 不支持，音效将被禁用');
            this.enabled = false;
        }
    }

    // 创建各种音效
    createSounds() {
        if (!this.enabled) return;

        // 合并音效 - 清脆的钟声
        this.sounds.merge = this.createMergeSound();
        
        // 移动音效 - 轻柔的滑动声
        this.sounds.move = this.createMoveSound();
        
        // 新方块出现音效
        this.sounds.newTile = this.createNewTileSound();
        
        // 胜利音效
        this.sounds.victory = this.createVictorySound();
        
        // 游戏结束音效
        this.sounds.gameOver = this.createGameOverSound();
    }

    // 创建合并音效 - 清脆的音调
    createMergeSound() {
        return (frequency = 800, duration = 0.2) => {
            if (!this.audioContext) return;
            
            const oscillator = this.audioContext.createOscillator();
            const gainNode = this.audioContext.createGain();
            
            oscillator.connect(gainNode);
            gainNode.connect(this.audioContext.destination);
            
            // 钟声效果 - 使用正弦波加上一些泛音
            oscillator.type = 'sine';
            oscillator.frequency.setValueAtTime(frequency, this.audioContext.currentTime);
            oscillator.frequency.exponentialRampToValueAtTime(frequency * 1.5, this.audioContext.currentTime + 0.05);
            oscillator.frequency.exponentialRampToValueAtTime(frequency, this.audioContext.currentTime + duration);
            
            // 音量包络
            gainNode.gain.setValueAtTime(0, this.audioContext.currentTime);
            gainNode.gain.linearRampToValueAtTime(this.volume * 0.3, this.audioContext.currentTime + 0.01);
            gainNode.gain.exponentialRampToValueAtTime(0.001, this.audioContext.currentTime + duration);
            
            oscillator.start(this.audioContext.currentTime);
            oscillator.stop(this.audioContext.currentTime + duration);
        };
    }

    // 创建移动音效 - 轻柔的滑动声
    createMoveSound() {
        return () => {
            if (!this.audioContext) return;
            
            const oscillator = this.audioContext.createOscillator();
            const gainNode = this.audioContext.createGain();
            const filter = this.audioContext.createBiquadFilter();
            
            oscillator.connect(filter);
            filter.connect(gainNode);
            gainNode.connect(this.audioContext.destination);
            
            oscillator.type = 'sawtooth';
            oscillator.frequency.setValueAtTime(200, this.audioContext.currentTime);
            oscillator.frequency.linearRampToValueAtTime(100, this.audioContext.currentTime + 0.1);
            
            filter.type = 'lowpass';
            filter.frequency.setValueAtTime(1000, this.audioContext.currentTime);
            
            gainNode.gain.setValueAtTime(0, this.audioContext.currentTime);
            gainNode.gain.linearRampToValueAtTime(this.volume * 0.1, this.audioContext.currentTime + 0.01);
            gainNode.gain.exponentialRampToValueAtTime(0.001, this.audioContext.currentTime + 0.1);
            
            oscillator.start(this.audioContext.currentTime);
            oscillator.stop(this.audioContext.currentTime + 0.1);
        };
    }

    // 创建新方块音效
    createNewTileSound() {
        return () => {
            if (!this.audioContext) return;
            
            const oscillator = this.audioContext.createOscillator();
            const gainNode = this.audioContext.createGain();
            
            oscillator.connect(gainNode);
            gainNode.connect(this.audioContext.destination);
            
            oscillator.type = 'triangle';
            oscillator.frequency.setValueAtTime(400, this.audioContext.currentTime);
            oscillator.frequency.linearRampToValueAtTime(600, this.audioContext.currentTime + 0.05);
            
            gainNode.gain.setValueAtTime(0, this.audioContext.currentTime);
            gainNode.gain.linearRampToValueAtTime(this.volume * 0.15, this.audioContext.currentTime + 0.01);
            gainNode.gain.exponentialRampToValueAtTime(0.001, this.audioContext.currentTime + 0.1);
            
            oscillator.start(this.audioContext.currentTime);
            oscillator.stop(this.audioContext.currentTime + 0.1);
        };
    }

    // 创建胜利音效
    createVictorySound() {
        return () => {
            if (!this.audioContext) return;
            
            // 播放一系列上升的音调
            const notes = [523, 659, 784, 1047]; // C, E, G, C
            notes.forEach((freq, index) => {
                setTimeout(() => {
                    this.playTone(freq, 0.3, 'sine', this.volume * 0.4);
                }, index * 100);
            });
        };
    }

    // 创建游戏结束音效
    createGameOverSound() {
        return () => {
            if (!this.audioContext) return;
            
            // 播放下降的音调
            const notes = [400, 350, 300, 250];
            notes.forEach((freq, index) => {
                setTimeout(() => {
                    this.playTone(freq, 0.5, 'sawtooth', this.volume * 0.2);
                }, index * 150);
            });
        };
    }

    // 通用音调播放函数
    playTone(frequency, duration, type = 'sine', volume = 0.3) {
        if (!this.audioContext || !this.enabled) return;
        
        const oscillator = this.audioContext.createOscillator();
        const gainNode = this.audioContext.createGain();
        
        oscillator.connect(gainNode);
        gainNode.connect(this.audioContext.destination);
        
        oscillator.type = type;
        oscillator.frequency.setValueAtTime(frequency, this.audioContext.currentTime);
        
        gainNode.gain.setValueAtTime(0, this.audioContext.currentTime);
        gainNode.gain.linearRampToValueAtTime(volume, this.audioContext.currentTime + 0.01);
        gainNode.gain.exponentialRampToValueAtTime(0.001, this.audioContext.currentTime + duration);
        
        oscillator.start(this.audioContext.currentTime);
        oscillator.stop(this.audioContext.currentTime + duration);
    }

    // 播放合并音效（根据方块值调整音调）
    playMerge(tileValue) {
        if (!this.enabled || !this.sounds.merge) return;
        
        // 根据方块值计算音调
        const baseFreq = 400;
        const multiplier = Math.log2(tileValue / 2) * 0.1 + 1;
        const frequency = Math.min(baseFreq * multiplier, 1200);
        
        this.sounds.merge(frequency, 0.25);
    }

    // 播放移动音效
    playMove() {
        if (!this.enabled || !this.sounds.move) return;
        this.sounds.move();
    }

    // 播放新方块音效
    playNewTile() {
        if (!this.enabled || !this.sounds.newTile) return;
        this.sounds.newTile();
    }

    // 播放胜利音效
    playVictory() {
        if (!this.enabled || !this.sounds.victory) return;
        this.sounds.victory();
    }

    // 播放游戏结束音效
    playGameOver() {
        if (!this.enabled || !this.sounds.gameOver) return;
        this.sounds.gameOver();
    }

    // 设置音量
    setVolume(volume) {
        this.volume = Math.max(0, Math.min(1, volume));
    }

    // 启用/禁用音效
    setEnabled(enabled) {
        this.enabled = enabled;
        if (enabled && !this.audioContext) {
            this.initAudioContext();
        }
    }

    // 恢复音频上下文（需要用户交互才能启动）
    resume() {
        if (this.audioContext && this.audioContext.state === 'suspended') {
            this.audioContext.resume();
        }
    }
}

// 全局音效管理器实例
window.audioManager = new AudioManager(); 