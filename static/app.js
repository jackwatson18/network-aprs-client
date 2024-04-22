const API_URL = "http://localhost:8080"

Vue.createApp({

data: function () {
    return {
        packets: [],
        myCallsign: "",
        mySSID: 0,
        myLat: 0,
        myLong: 0,
        myComment: ""


    };
},

methods: {
    getPacketsFromServer: function() {
        fetch(API_URL+"/packets").then(response => {
            response.json().then( data => {
                console.log("Loaded packets from server:", data);
                this.packets = data;
            })
        })
    },

    sendPositionPacketToServer: function() {
        var data = {
            Callsign: this.myCallsign,
            Ssid: this.mySSID,
            Latitude: this.myLat,
            Longitude: this.myLong,
            Comment: this.myComment
        }

        data = JSON.stringify(data)
        console.log(data)

        fetch(API_URL+"/packets", {
            method: "POST",
            body: data,
            headers: {
                "Context-Type": "application/json"
            }
        }).then( response => {
            if (response.status == 201) {
                console.log("Send okay!")
            }
        });
    },

    sendMessagePacketToServer: function() {
        var data = {
                Callsign: this.myCallsign,
                Ssid: this.mySSID,
                Latitude: 0,
                Longitude: 0,
                Comment: ":"+this.myComment
            }

            data = JSON.stringify(data)
            console.log(data)

            fetch(API_URL+"/packets", {
                method: "POST",
                body: data,
                headers: {
                    "Context-Type": "application/json"
                }
            }).then( response => {
                if (response.status == 201) {
                    console.log("Send okay!")
                }
            });

        
    }
},

created: function() {
    this.getPacketsFromServer();
    this.timer = setInterval(this.getPacketsFromServer, 10000)
}

}).mount("#app");