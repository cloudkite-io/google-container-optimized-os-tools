#!/bin/bash
main() {
    cat <<EOF
        Architecture:        x86_64
        CPU op-mode(s):      32-bit, 64-bit
        Byte Order:          Little Endian
        Address sizes:       39 bits physical, 48 bits virtual
        CPU(s):              8
        On-line CPU(s) list: 0-7
        Thread(s) per core:  1
        Core(s) per socket:  8
        Socket(s):           1
        Vendor ID:           GenuineIntel
        CPU family:          6
        Model:               142
        Model name:          06/8e
        Stepping:            12
        CPU MHz:             2303.980
        BogoMIPS:            4607.96
        Virtualization:      VT-x
        Hypervisor vendor:   KVM
        Virtualization type: full
        L1d cache:           32K
        L1i cache:           32K
        L2 cache:            256K
        L3 cache:            8192K
        Flags:               fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ss ht syscall nx pdpe1gb rdtscp lm constant_tsc arch_perfmon rep_good nopl xtopology nonstop_tsc cpuid tsc_known_freq pni pclmulqdq vmx ssse3 fma cx16 pcid sse4_1 sse4_2 x2apic movbe popcnt tsc_deadline_timer aes xsave avx f16c rdrand hypervisor lahf_lm abm 3dnowprefetch cpuid_fault invpcid_single ssbd ibrs ibpb stibp ibrs_enhanced tpr_shadow vnmi flexpriority ept vpid ept_ad fsgsbase tsc_adjust bmi1 avx2 smep bmi2 erms invpcid mpx rdseed adx smap clflushopt xsaveopt xsavec xgetbv1 xsaves arat umip md_clear arch_capabilities
EOF
}
main "$#"
