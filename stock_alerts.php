# stock_alerts.php
#!/usr/bin/env php
<?php

define('CONFIG_FILE', 'alerts.json');
define('API_KEY', 'demo');
define('BASE_URL', 'https://www.alphavantage.co/query');

class Alert {
    public $id;
    public $symbol;
    public $threshold;
    public $above;
    public $triggered;

    function __construct($symbol, $threshold, $above = true, $id = null) {
        $this->id = $id ?: substr(bin2hex(random_bytes(4)), 0, 8);
        $this->symbol = strtoupper($symbol);
        $this->threshold = (float)$threshold;
        $this->above = $above;
        $this->triggered = false;
    }

    function toArray() {
        return [
            'id' => $this->id,
            'symbol' => $this->symbol,
            'threshold' => $this->threshold,
            'above' => $this->above,
            'triggered' => $this->triggered
        ];
    }

    static function fromArray($data) {
        $a = new self($data['symbol'], $data['threshold'], $data['above'], $data['id']);
        $a->triggered = $data['triggered'] ?? false;
        return $a;
    }

    function check($price) {
        $triggered = $this->above ? $price > $this->threshold : $price < $this->threshold;
        if ($triggered && !$this->triggered) {
            $this->triggered = true;
            return true;
        }
        return false;
    }
}

class App {
    private $alerts = [];

    function __construct() {
        $this->load();
    }

    function load() {
        if (file_exists(CONFIG_FILE)) {
            $data = json_decode(file_get_contents(CONFIG_FILE), true);
            foreach ($data as $item) {
                $this->alerts[] = Alert::fromArray($item);
            }
        }
    }

    function save() {
        $data = array_map(function($a) { return $a->toArray(); }, $this->alerts);
        file_put_contents(CONFIG_FILE, json_encode($data, JSON_PRETTY_PRINT));
    }

    function add($symbol, $threshold, $above = true) {
        $this->alerts = array_filter($this->alerts, function($a) use ($symbol) {
            return $a->symbol != strtoupper($symbol);
        });
        $alert = new Alert($symbol, $threshold, $above);
        $this->alerts[] = $alert;
        $this->save();
        echo "✅ Added alert: $symbol " . ($above ? '>' : '<') . " " . number_format($threshold, 2) . "\n";
    }

    function remove($symbol) {
        $this->alerts = array_filter($this->alerts, function($a) use ($symbol) {
            return $a->symbol != strtoupper($symbol);
        });
        $this->save();
        echo "✅ Removed alert for " . strtoupper($symbol) . "\n";
    }

    function list() {
        if (empty($this->alerts)) {
            echo "No alerts.\n";
            return;
        }
        echo "\n📋 Current alerts:\n";
        foreach ($this->alerts as $a) {
            $status = $a->triggered ? '🔔' : '⏳';
            echo "  {$a->symbol}: " . ($a->above ? '>' : '<') . " " . number_format($a->threshold, 2) . " $status\n";
        }
    }

    function fetchPrice($symbol) {
        $url = BASE_URL . "?function=GLOBAL_QUOTE&symbol=$symbol&apikey=" . API_KEY;
        $resp = file_get_contents($url);
        if ($resp === false) return null;
        $data = json_decode($resp, true);
        if (!isset($data['Global Quote'])) return null;
        return (float)$data['Global Quote']['05. price'];
    }

    function checkAlerts() {
        foreach ($this->alerts as $alert) {
            $price = $this->fetchPrice($alert->symbol);
            if ($price === null) continue;
            if ($alert->check($price)) {
                $direction = $alert->above ? 'above' : 'below';
                echo "\n🔔 ALERT: {$alert->symbol} price is " . number_format($price, 2) . " ($direction " . number_format($alert->threshold, 2) . ")!\n";
                $this->save();
            }
        }
    }

    function watch($interval = 60) {
        echo "📈 Monitoring alerts every {$interval}s (press Ctrl+C to stop)\n";
        while (true) {
            $this->checkAlerts();
            sleep($interval);
        }
    }
}

if ($argc < 2) {
    die("Usage: php stock_alerts.php <command> [options]\n");
}
$app = new App();
$cmd = $argv[1];

switch ($cmd) {
    case 'add':
        if ($argc < 4) die("add <symbol> <threshold> [--above] [--below]\n");
        $symbol = $argv[2];
        $threshold = (float)$argv[3];
        $above = true;
        for ($i=4; $i<$argc; $i++) {
            if ($argv[$i] == '--below') $above = false;
        }
        $app->add($symbol, $threshold, $above);
        break;
    case 'remove':
        if ($argc < 3) die("remove <symbol>\n");
        $app->remove($argv[2]);
        break;
    case 'list':
        $app->list();
        break;
    case 'watch':
        $interval = 60;
        for ($i=2; $i<$argc; $i++) {
            if ($argv[$i] == '--interval' && isset($argv[$i+1])) {
                $interval = (int)$argv[++$i];
            }
        }
        $app->watch($interval);
        break;
    default:
        echo "Unknown command.\n";
}
?>
