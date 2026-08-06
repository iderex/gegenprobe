# gegenprobe

The atomic data behind stellar atmospheres, fusion plasmas and X-ray spectra come from a handful of Fortran codes whose results diverge, and the standard uncertainty estimate is to run three or four and call the scatter the error. In the lithium-like sequence at high Z the methods differed by up to 40 eV for U-89+, and it took a dedicated publication to resolve it to 2 eV against experiment. Cowans package, four Fortran programs in series from 1981, still gains 150 citations a year because modern ab initio methods do not reproduce complex heavy atoms. Two components: a harness taking a declarative input, running all codes containerised and comparing them automatically to produce a map of where methods agree, and a replacement for the interactive Slater-parameter least-squares fit that actually produces most published atomic data.

Planning happens on the issue tracker first. Every decision that shapes
the architecture is written down there with its reasons before the code
that depends on it exists.

See [NOTICE.md](NOTICE.md) for the intended-use notice.
