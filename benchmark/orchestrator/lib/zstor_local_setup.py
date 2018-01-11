import os
import shutil
import subprocess
import tempfile
from random import randint


# SetupZstor is responsible for managing a zstor setup

Base = '127.0.0.1' # base of addresses at the local host

class SetupZstor:

    def __init__(self, ):
        self.zstor_nodes = []
        self.etcd_nodes = []
        self.cleanup_dirs = []
        self.data_shards = []
        self.meta_shards = []

    # start zstordb servers
    def run_zstordb_servers(self, servers=2, no_auth=True, jobs=0, start_port=1200, profile=None, profile_dir="profile", data_dir=None, meta_dir=None):
        for i in range(0, servers):
            if not data_dir:
                db_dir = tempfile.mkdtemp()
            else:
                db_dir = os.path.join(data_dir, str(i))
                os.makedirs(data_dir)

            if not meta_dir:
                md_dir = tempfile.mkdtemp()
            else:
                md_dir = os.path.join(meta_dir, str(i))
                os.makedirs(meta_dir)

            self.cleanup_dirs.extend((db_dir, md_dir))

            port = str(start_port + i)
            self.data_shards.append('%s:%s'%(Base,port))

            args = ["zstordb",
                    "--listen", ":" + port,
                    "--data-dir", db_dir,
                    "--meta-dir", md_dir,
                    "--jobs", str(jobs),
                    ]

            if profile and is_profile_flag(profile):
                args.extend(("--profile-mode", profile))

                profile_dir_zstordb = os.path.join(profile_dir, "zstordb" + str(i))

                if not os.path.exists(profile_dir_zstordb):
                    os.makedirs(profile_dir_zstordb)

                args.extend(("--profile-output", profile_dir_zstordb))

            if no_auth:
                args.append("--no-auth")

            self.zstor_nodes.append(subprocess.Popen(args, stderr=subprocess.PIPE))

    # stop zstordb servers
    def stop_zstordb_servers(self):
        self.data_shards = []
        for node in self.zstor_nodes:
            node.terminate()
            _, err = node.communicate()
            # apparently resturned code is positive when failed (2 or 255)
            # and -15 when successfully terminated
            if node.returncode > 0:
                print("zstor server exited with code %d:" % node.returncode)
                print(err.decode())
                print()

        self.zstor_nodes = []

    # start etcd servers
    # with client port start__port + server number
    # with peer port start__port + server number + 100
    def run_etcd_servers(self, servers=2, start_port=1300,  data_dir=""):
        cluster_token = "etcd-cluster-" + str(randint(0, 99))
        names = []
        peer_addresses = []
        client_addresses = []
        init_cluster = ""
        base = "http://127.0.0.1:"

        for i in range(0, servers):
            name = "node" + str(i)

            port = str(start_port + i)
            self.meta_shards.append('%s:%s'%(Base,port))

            client_port = base + port
            peer_port = base + str(start_port + 100 + i)
            init_cluster += name + "=" + peer_port + ","

            names.append(name)
            peer_addresses.append(peer_port)
            client_addresses.append(client_port)

        for i in range(0, servers):
            name = names[i]
            client_address = client_addresses[i]
            peer_address = peer_addresses[i]

            if data_dir == "":
                db_dir = tempfile.mkdtemp()
            else:
                db_dir = data_dir + "/etcd" + str(i)

            self.cleanup_dirs.append(db_dir)

            args = ["etcd",
                    "--name", name,
                    "--initial-advertise-peer-urls", peer_address,
                    "--listen-peer-urls", peer_address,
                    "--listen-client-urls", client_address,
                    "--advertise-client-urls", client_address,
                    "--initial-cluster-token", cluster_token,
                    "--initial-cluster", init_cluster,
                    "--data-dir", db_dir,
                    ]

            self.etcd_nodes.append(subprocess.Popen(args, stdout=subprocess.PIPE, stderr=subprocess.PIPE))

    # stop etcd servers
    def stop_etcd_servers(self):
        self.meta_shards = []
        for node in self.etcd_nodes:
            node.terminate()
            _, err = node.communicate()

            # apparently resturned code is positive when failed (2 or 255)
            # and -15 when successfully terminated
            if node.returncode > 0:
                print("etcd server exited with code %d:" % node.returncode)
                print(err.decode())
                print()

        self.etcd_nodes = []

    def cleanup(self):
        while self.cleanup_dirs:
            shutil.rmtree(self.cleanup_dirs.pop())

# returns true if provided profile flag is valid
def is_profile_flag(flag):
    return flag in ('cpu', 'mem', 'block', 'trace')


if __name__ == '__main__':
    z = SetupZstor()
    from IPython import embed
    embed()
