string queryRequestURL = "[Insert your full URL here]";
integer listener = 0;
integer CHAT_CHANNEL = 5;
key http_request_id;
key avatar;

default
{
	state_entry()
	{
		llSetText("Touch to get an avatar UUID from Second Life®", <1,1,1>, 1);
	}

	touch_start(integer howmany)
	{
		if (listener == 0)
		{
			avatar = llDetectedKey(0);
			listener = llListen(CHAT_CHANNEL, "", avatar, "");
			llSetText("In use by " + llDetectedName(0), <1,1,1>,1);
			llWhisper(0, "Say an avatar name in chat prefixing it with /" + (string)CHAT_CHANNEL + "; touch to reset");
			llSetTimerEvent(60.0);
		}
		else if (avatar == llDetectedKey(0))
		{
			llResetScript();
		}
		else
			llWhisper(0, "In use");
	}

	timer()
	{
		llResetScript();
	}

	listen(integer channel, string name, key id, string message)
	{
		http_request_id = llHTTPRequest(queryRequestURL + "?name=" + llEscapeURL(message) + "&compat=false", [], "");
		llSetTimerEvent(60.0);
	}

	http_response(key request_id, integer status, list metadata, string body)
	{
		if (request_id == http_request_id)
		{
			if (status == 200)
				llInstantMessage(avatar, body);
			else
				llInstantMessage(avatar, "Error " + (string)status + ": " + body);
		}
		llSetTimerEvent(0.0);
	}
}