import os
from seedemu.core import Node, Service, Server
from typing import Dict

WebServerFileTemplates: Dict[str, str] = {}

WebServerFileTemplates['nginx_site'] = '''\
server {{
    listen {port};
    root /var/www/html;
    index index.html;
    server_name _;
    location / {{
        try_files $uri $uri/ =404;
    }}
    location /images/ {{
        alias /var/www/content/images/;
    }}
    location /videos/ {{
        alias /var/www/content/videos/;
    }}
}}
'''

class ScionWebServer(Server):
    def __init__(self):
        super().__init__()

        self.__port = 8080
        self.__index = '<h1>{nodeName} at {asn}</h1>'
        self.__text_dir = '/var/www/content/text/'
        self.__image_dir = '/var/www/content/images/'
        self.__video_dir = '/var/www/content/videos/'

    def setPort(self, port: int) -> 'ScionWebServer':
        self.__port = port
        return self

    def setIndexContent(self, content: str) -> 'ScionWebServer':
        self.__index = content
        return self
    
    def setImageDir(self, image_dir: str) -> 'ScionWebServer':
        self.__image_dir = image_dir
        return self
    
    def setVideoDir(self, video_dir: str) -> 'ScionWebServer':
        self.__video_dir = video_dir
        return self
    
    def install(self, node: Node):
        node.addSoftware('nginx-light')
        node.setFile('/var/www/html/index.html', self.__index.format(asn=node.getAsn(), nodeName=node.getName()))
        node.setFile('/etc/nginx/sites-available/default', WebServerFileTemplates['nginx_site'].format(port=self.__port))
        node.appendStartCommand('service nginx start')

        # Create content directories if they don't exist
        for directory in [self.__text_dir, self.__image_dir, self.__video_dir]:
            node.appendStartCommand(f'mkdir -p {directory}')

        node.appendClassName("ScionWebServer")

    def print(self, indent: int) -> str:
        out = ' ' * indent
        out += 'Scion Web server object.\n'
        return out

class ScionWebService(Service):
    def __init__(self):
        super().__init__()
        self.addDependency('Base', False, False)

    def _createServer(self) -> Server:
        return ScionWebServer()

    def getName(self) -> str:
        return 'ScionWebService'

    def print(self, indent: int) -> str:
        out = ' ' * indent
        out += 'Scion WebService Layer\n'
        return out
