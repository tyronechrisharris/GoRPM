import sys
import threading
import time
sys.path.append('/usr/lib/python3/dist-packages')
import gi
gi.require_version('Gst', '1.0')
gi.require_version('GstRtspServer', '1.0')
from gi.repository import Gst, GstRtspServer, GLib
Gst.init(None)

class RTSPServer:
    def __init__(self, port: str = "8554"):
        self.port = port
        self.server = GstRtspServer.RTSPServer()
        self.server.set_service(self.port)
        self.factory = GstRtspServer.RTSPMediaFactory()

        # Testing pipeline with clockoverlay but no textoverlay
        pipeline = (
            "( videotestsrc pattern=black ! "
            "video/x-raw,width=640,height=480,framerate=15/1 ! "
            "clockoverlay time-format=\"%H:%M:%S.%%06u\" ! "
            "x264enc speed-preset=ultrafast tune=zerolatency ! "
            "rtph264pay name=pay0 pt=96 )"
        )
        self.factory.set_launch(pipeline)
        self.factory.set_shared(True)
        self.server.get_mount_points().add_factory("/camera", self.factory)

        self._running = False

    def start(self):
        if self._running:
            return

        self.server.attach(None)
        self._running = True

        if not hasattr(RTSPServer, "_global_loop"):
            RTSPServer._global_loop = GLib.MainLoop()
            RTSPServer._global_thread = threading.Thread(target=RTSPServer._global_loop.run, name="RTSP-GlobalLoop")
            RTSPServer._global_thread.daemon = True
            RTSPServer._global_thread.start()

srv1 = RTSPServer("8554")
srv1.start()
time.sleep(10)
