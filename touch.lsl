string touchURL = "[Insert your full URL here]/touch/";
key http_request_id;

default
{
    state_entry()
    {
        llSetText("Touch to register your avatar name and UUID", <1,1,1>, 1);
    }
    
    touch_start(integer howmany)
    {
        integer i;
        llSetText("Sending...", <1,0,0>, 1);
        for (i = 0; i < howmany; i++) {
            http_request_id = llHTTPRequest(touchURL + "?name=" + llEscapeURL(llDetectedName(i)) +
                "&amp;key=" + llEscapeURL(llDetectedKey(i)), [], "");
            llSetTimerEvent(360.0);   
        }
        llSetText("Touch to register your avatar name and UUID", <1,1,1>, 1);
    }
    
    timer()
    {
        llWhisper(0, "No response from web services...");
        llResetScript();
    }

    http_response(key request_id, integer status, list metadata, string body)
    {
        if (request_id == http_request_id)
        {
            if (status == 200)
                llWhisper(0, body);
            else
                llWhisper(0, "Error " + (string)status + ": " + body);
            llSetTimerEvent(0.0);
        }
    }
}