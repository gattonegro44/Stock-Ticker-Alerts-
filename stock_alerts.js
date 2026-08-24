// stock_alerts.js
#!/usr/bin/env node
const fs = require('fs');
const axios = require('axios');
const { program } = require('commander');
const { v4: uuidv4 } = require('uuid');

const CONFIG_FILE = 'alerts.json';
const API_KEY = 'demo';
const BASE_URL = 'https://www.alphavantage.co/query';

class Alert {
    constructor(symbol, threshold, above = true, id = null) {
        this.id = id || uuidv4().slice(0,8);
        this.symbol = symbol.toUpperCase();
        this.threshold = threshold;
        this.above = above;
        this.triggered = false;
    }

    check(price) {
        const triggered = this.above ? price > this.threshold : price < this.threshold;
        if (triggered && !this.triggered) {
            this.triggered = true;
            return true;
        }
        return false;
    }
}

class App {
    constructor() {
        this.alerts = [];
        this.load();
    }

    load() {
        if (fs.existsSync(CONFIG_FILE)) {
            try {
                const data = JSON.parse(fs.readFileSync(CONFIG_FILE));
                this.alerts = data.map(a => {
                    const alert = new Alert(a.symbol, a.threshold, a.above, a.id);
                    alert.triggered = a.triggered || false;
                    return alert;
                });
            } catch (e) {}
        }
    }

    save() {
        fs.writeFileSync(CONFIG_FILE, JSON.stringify(this.alerts.map(a => ({
            id: a.id,
            symbol: a.symbol,
            threshold: a.threshold,
            above: a.above,
            triggered: a.triggered
        })), null, 2));
    }

    add(symbol, threshold, above = true) {
        // Remove existing
        this.alerts = this.alerts.filter(a => a.symbol !== symbol.toUpperCase());
        const alert = new Alert(symbol, threshold, above);
        this.alerts.push(alert);
        this.save();
        console.log(`✅ Added alert: ${symbol} ${above ? '>' : '<'} ${threshold.toFixed(2)}`);
    }

    remove(symbol) {
        this.alerts = this.alerts.filter(a => a.symbol !== symbol.toUpperCase());
        this.save();
        console.log(`✅ Removed alert for ${symbol.toUpperCase()}`);
    }

    list() {
        if (this.alerts.length === 0) {
            console.log('No alerts.');
            return;
        }
        console.log('\n📋 Current alerts:');
        for (const a of this.alerts) {
            const status = a.triggered ? '🔔' : '⏳';
            console.log(`  ${a.symbol}: ${a.above ? '>' : '<'} ${a.threshold.toFixed(2)} ${status}`);
        }
    }

    async fetchPrice(symbol) {
        const url = `${BASE_URL}?function=GLOBAL_QUOTE&symbol=${symbol}&apikey=${API_KEY}`;
        try {
            const resp = await axios.get(url, { timeout: 10000 });
            const quote = resp.data['Global Quote'];
            if (!quote) return null;
            return parseFloat(quote['05. price']);
        } catch (e) {
            console.error(`Error fetching ${symbol}: ${e.message}`);
            return null;
        }
    }

    async checkAlerts() {
        for (const alert of this.alerts) {
            const price = await this.fetchPrice(alert.symbol);
            if (price === null) continue;
            if (alert.check(price)) {
                const direction = alert.above ? 'above' : 'below';
                console.log(`\n🔔 ALERT: ${alert.symbol} price is ${price.toFixed(2)} (${direction} ${alert.threshold.toFixed(2)})!`);
                this.save();
            }
        }
    }

    async watch(interval = 60) {
        console.log(`📈 Monitoring alerts every ${interval}s (press Ctrl+C to stop)`);
        while (true) {
            await this.checkAlerts();
            await new Promise(resolve => setTimeout(resolve, interval * 1000));
        }
    }
}

program
    .command('add <symbol> <threshold>')
    .option('--above', 'Alert when price goes above threshold', true)
    .option('--below', 'Alert when price goes below threshold')
    .action((symbol, threshold, options) => {
        const app = new App();
        const above = !options.below;
        app.add(symbol, parseFloat(threshold), above);
    });

program
    .command('remove <symbol>')
    .action((symbol) => {
        const app = new App();
        app.remove(symbol);
    });

program
    .command('list')
    .action(() => {
        const app = new App();
        app.list();
    });

program
    .command('watch')
    .option('--interval <n>', 'Check interval in seconds', parseInt, 60)
    .action(async (options) => {
        const app = new App();
        await app.watch(options.interval);
    });

program.parse(process.argv);
