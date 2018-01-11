import lib.zstor_local_setup as zstor
import subprocess

class Bench_client:
    def __init__(self, ):
        pass
    
    def run(self, profile=None, profile_dir="profile_client", config="client_config.yaml", out="bench_result.yaml"):

        args = ["zstorbench",
            "--conf", config,
            "--out-benchmark", out,
            ]
                    
        if profile and zstor.is_profile_flag(profile):
            args.extend(("--profile-mode", profile))
            args.extend(("--out-profile", profile_dir))

        # run benchmark client
        subprocess.run(args, )
