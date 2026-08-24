// stock_alerts.cpp
#include <iostream>
#include <fstream>
#include <string>
#include <vector>
#include <map>
#include <thread>
#include <chrono>
#include <curl/curl.h>
#include <nlohmann/json.hpp>

using namespace std;
using json = nlohmann::json;

const string CONFIG_FILE = "alerts.json";
const string API_KEY = "demo";
const string BASE_URL = "https://www.alphavantage.co/query";

size_t WriteCallback(void *contents, size_t size, size_t nmemb, void *userp) {
    ((string*)userp)->append((char*)contents, size * nmemb);
    return size * nmemb;
}

string fetchUrl(const string& url) {
    CURL *curl = curl_easy_init();
    string response;
    if (curl) {
        curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteCallback);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response);
        curl_easy_setopt(curl, CURLOPT_TIMEOUT, 10L);
        CURLcode res = curl_easy_perform(curl);
        curl_easy_cleanup(curl);
        if (res != CURLE_OK) return "";
    }
    return response;
}

struct Alert {
    string id;
    string symbol;
    double threshold;
    bool above;
    bool triggered;

    Alert() : threshold(0), above(true), triggered(false) {}
    Alert(const string& sym, double th, bool abv) 
        : id(generateId()), symbol(sym), threshold(th), above(abv), triggered(false) {}

    static string generateId() {
        const char* hex = "0123456789abcdef";
        string id;
        random_device rd;
        mt19937 gen(rd());
        uniform_int_distribution<> dis(0, 15);
        for (int i=0; i<8; i++) id += hex[dis(gen)];
        return id;
    }
};

class App {
private:
    vector<Alert> alerts;
    string configFile;

    void load() {
        ifstream f(configFile);
        if (!f.is_open()) return;
        json j;
        f >> j;
        for (auto& item : j) {
            Alert a;
            a.id = item["id"];
            a.symbol = item["symbol"];
            a.threshold = item["threshold"];
            a.above = item["above"];
            a.triggered = item["triggered"];
            alerts.push_back(a);
        }
    }

    void save() {
        json j = json::array();
        for (auto& a : alerts) {
            j.push_back({{"id", a.id}, {"symbol", a.symbol}, {"threshold", a.threshold}, {"above", a.above}, {"triggered", a.triggered}});
        }
        ofstream f(configFile);
        f << setw(2) << j << endl;
    }

    double fetchPrice(const string& symbol) {
        string url = BASE_URL + "?function=GLOBAL_QUOTE&symbol=" + symbol + "&apikey=" + API_KEY;
        string resp = fetchUrl(url);
        if (resp.empty()) return 0;
        auto data = json::parse(resp);
        if (!data.contains("Global Quote")) return 0;
        string priceStr = data["Global Quote"]["05. price"];
        return stod(priceStr);
    }

    Alert* getAlert(const string& symbol) {
        for (auto& a : alerts) {
            if (a.symbol == symbol) return &a;
        }
        return nullptr;
    }

public:
    App() : configFile(CONFIG_FILE) { load(); }

    void add(const string& symbol, double threshold, bool above) {
        Alert* existing = getAlert(symbol);
        if (existing) {
            alerts.erase(remove_if(alerts.begin(), alerts.end(), [&](const Alert& a){ return a.symbol == symbol; }), alerts.end());
        }
        Alert a(symbol, threshold, above);
        alerts.push_back(a);
        save();
        cout << "✅ Added alert: " << symbol << " " << (above ? ">" : "<") << " " << fixed << setprecision(2) << threshold << "\n";
    }

    void remove(const string& symbol) {
        alerts.erase(remove_if(alerts.begin(), alerts.end(), [&](const Alert& a){ return a.symbol == symbol; }), alerts.end());
        save();
        cout << "✅ Removed alert for " << symbol << "\n";
    }

    void list() {
        if (alerts.empty()) {
            cout << "No alerts.\n";
            return;
        }
        cout << "\n📋 Current alerts:\n";
        for (auto& a : alerts) {
            string status = a.triggered ? "🔔" : "⏳";
            cout << "  " << a.symbol << ": " << (a.above ? ">" : "<") << " " << fixed << setprecision(2) << a.threshold << " " << status << "\n";
        }
    }

    void checkAlerts() {
        for (auto& a : alerts) {
            double price = fetchPrice(a.symbol);
            if (price == 0) continue;
            bool triggered = a.above ? price > a.threshold : price < a.threshold;
            if (triggered && !a.triggered) {
                a.triggered = true;
                cout << "\n🔔 ALERT: " << a.symbol << " price is " << fixed << setprecision(2) << price << " (" << (a.above ? "above" : "below") << " " << a.threshold << ")!\n";
                save();
            }
        }
    }

    void watch(int interval) {
        cout << "📈 Monitoring alerts every " << interval << "s (press Ctrl+C to stop)\n";
        while (true) {
            checkAlerts();
            this_thread::sleep_for(chrono::seconds(interval));
        }
    }
};

int main(int argc, char* argv[]) {
    if (argc < 2) {
        cerr << "Usage: stock_alerts <command> [options]\n";
        return 1;
    }
    App app;
    string cmd = argv[1];

    if (cmd == "add") {
        if (argc < 4) { cerr << "add <symbol> <threshold> [--above] [--below]\n"; return 1; }
        string symbol = argv[2];
        double threshold = stod(argv[3]);
        bool above = true;
        for (int i=4; i<argc; i++) {
            if (string(argv[i]) == "--below") above = false;
        }
        app.add(symbol, threshold, above);
    } else if (cmd == "remove") {
        if (argc < 3) { cerr << "remove <symbol>\n"; return 1; }
        app.remove(argv[2]);
    } else if (cmd == "list") {
        app.list();
    } else if (cmd == "watch") {
        int interval = 60;
        for (int i=2; i<argc; i++) {
            if (string(argv[i]) == "--interval" && i+1 < argc) {
                interval = stoi(argv[++i]);
            }
        }
        app.watch(interval);
    } else {
        cerr << "Unknown command.\n";
        return 1;
    }
    return 0;
}
