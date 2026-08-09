// Package model holds the types every reader produces and every later stage
// consumes, and the schema that describes the bundle those types are written
// into.
//
// It implements 0004, which fixes the three parts of the bundle and the
// canonical unit of every quantity; 0008, which requires a significance count
// and a marker on every number; 0011, which fixes the four states a cell can be
// in; and the layout 0007 gives the bundle on disk.
//
// Two properties are the reason this package exists rather than each reader
// carrying its own shape.
//
// A number with physical meaning is a [Quantity] and never a bare float. That
// type carries the unit and the significance count with the value, so a reader
// cannot produce a number that has lost either, and this package's own suite
// walks the type set and asserts the model holds no other float anywhere.
//
// A cell is never absent by being empty. Absence is a state and a reason from
// 0011, refused by the type on the way out and on the way in, so a blank that
// could mean any of four things cannot reach an artefact.
//
// The schema is generated from these types by [Schema] rather than maintained
// beside them. A schema kept by hand drifts against the types the moment
// somebody adds a field, and the drift is invisible until a consumer written
// against the schema meets a bundle written by the types.
//
// The observed level energies of [Observed] are in the model and are populated
// by nothing in this release. They are here because 0013 keeps the fit a
// separate component and this is the shape it will read, and putting them in
// later would mean a bundle format bump for every stored result.
//
// What is not here: the conversion to the canonical unit, which is #50, and the
// writing of a bundle to disk with its checksums, which is #54. This package is
// the shape both of them work in.
package model

// Format is the bundle format version, the `bundle-format` field 0007 puts in
// the manifest. It is a single positive integer, bumped by any change that can
// make a previously valid bundle invalid, change what a field means, or move
// the canonical bytes of a bundle that did not itself change.
//
// Every released version stays readable indefinitely, which 0004 makes stricter
// here than for a case file: a bundle is an archival artefact that may be
// attached to a publication and reread by somebody with no way to regenerate
// it, so dropping a version would break a citation.
const Format = 1

// Parity is even or odd. It is an enumerated value rather than a sign on
// another quantity, which 0004 requires, because a sign is a second meaning
// riding on a number and it is lost by the first route that takes an absolute
// value.
type Parity string

const (
	Even Parity = "even"
	Odd  Parity = "odd"
)

var parities = []Parity{Even, Odd}

// Multipole is the kind of radiative transition, electric or magnetic.
type Multipole string

const (
	Electric Multipole = "E"
	Magnetic Multipole = "M"
)

var multipoles = []Multipole{Electric, Magnetic}

// Gauge is which gauge a strength was computed in. Two gauges are two numbers
// and the model keeps them apart: a code producing both has produced two
// statements about one transition, and averaging them would be this project
// concluding something it has decided not to conclude.
type Gauge string

const (
	Length      Gauge = "length"
	Velocity    Gauge = "velocity"
	Unspecified Gauge = "unspecified"
)

var gauges = []Gauge{Length, Velocity, Unspecified}

// Medium is what a wavelength was quoted in. A code that printed an air
// wavelength keeps it, marked, because converting to vacuum needs a refractive
// index formula and which formula was used is a claim about the number that has
// to travel with it.
type Medium string

const (
	Vacuum Medium = "vacuum"
	Air    Medium = "air"
)

var media = []Medium{Vacuum, Air}

// Level is one energy level as one code produced it.
type Level struct {
	// ID is this model's identity for the level, unique inside one
	// participant's table. A transition names it rather than the code's own
	// index, which 0004 requires so that a renumbering inside a code does not
	// silently repoint a transition.
	ID string `json:"id"`
	// Index is the code's own index, kept because a reader checking against
	// the original file needs it and because it is not an identity.
	Index int `json:"index"`
	// Label is the code's own label, kept verbatim and unparsed. Nothing in
	// this package normalises it: label translation is where a bench silently
	// invents agreement, and 0004 puts it in the identification step where it
	// is visible and allowed to fail.
	Label string `json:"label"`
	// TwoJ is twice the total angular momentum, stored as an exact integer
	// numerator over two rather than as a float. J is a label, matching on it
	// has to be exact, and a floating point 0.5 arriving from a text field as
	// 0.49999999 turns an exact label into an approximate one.
	TwoJ int `json:"two-j"`
	// Parity is even or odd.
	Parity Parity `json:"parity"`
	// LeadingConfiguration is the code's own spelling of the configuration
	// carrying the largest weight, or empty where the code stated none.
	LeadingConfiguration string `json:"leading-configuration"`
	// MixingWeight is that configuration's weight, from zero to one. It is
	// carried because it is the only honest handle for identification across
	// codes: a level that is 51 per cent one configuration and a level that is
	// 94 per cent the same configuration are different objects for matching.
	MixingWeight Quantity `json:"mixing-weight"`
	// Energy is the energy relative to the ground level, which is what the
	// comparison reads.
	Energy Quantity `json:"energy"`
	// TotalEnergy is the code's own total energy, kept because it is what a
	// reader would check against the original output.
	TotalEnergy Quantity `json:"total-energy"`
	// Observed is the measured energy for this level, populated by nothing in
	// this release.
	Observed Observed `json:"observed"`
}

// Observed is an observed level energy with its source. Nothing in this release
// populates it, so both cells are absent with a reason saying the case did not
// ask. It exists now so that the fit component of 0013 does not force a bundle
// format bump on every stored result when it arrives.
type Observed struct {
	// Energy is the observed energy, in the canonical unit.
	Energy Quantity `json:"energy"`
	// Uncertainty is the stated uncertainty on it.
	Uncertainty Quantity `json:"uncertainty"`
	// Source names where the observation came from, and is empty while
	// nothing populates these cells.
	Source string `json:"source"`
}

// Transition is one radiative transition as one code produced it.
type Transition struct {
	// Upper and Lower name levels by their identity in this model, never by a
	// code's own index alone.
	Upper string `json:"upper"`
	Lower string `json:"lower"`
	// Multipole is electric or magnetic and Order is its order, so an E1 is
	// Electric with order one. Both are exact labels and carry no
	// significance.
	Multipole Multipole `json:"multipole"`
	Order     int       `json:"order"`
	// Energy is the transition energy in the canonical unit.
	Energy Quantity `json:"energy"`
	// Wavelength is the wavelength, and Medium says whether the code quoted it
	// in vacuum or in air.
	Wavelength Quantity `json:"wavelength"`
	Medium     Medium   `json:"medium"`
	// Strengths holds one entry per gauge the code produced, in the order the
	// gauges are listed here. A code producing both gauges produces two
	// entries and this model never reduces them to one.
	Strengths []Strength `json:"strengths"`
}

// Strength is what a code said about a transition's strength in one gauge.
type Strength struct {
	Gauge Gauge `json:"gauge"`
	// WeightedOscillatorStrength is gf, dimensionless. Where a code printed f
	// alone the reader multiplies by the lower level's statistical weight, and
	// records that it did.
	WeightedOscillatorStrength Quantity `json:"weighted-oscillator-strength"`
	// TransitionProbability is A, in the canonical unit.
	TransitionProbability Quantity `json:"transition-probability"`
	// LineStrength is S, carried only where the code printed one.
	LineStrength Quantity `json:"line-strength"`
}

// Result is one participant's tables, the content of one
// `bundle/result/<participant>.json`.
type Result struct {
	// Participant is the participant identifier from the case.
	Participant string       `json:"participant"`
	Levels      []Level      `json:"levels"`
	Transitions []Transition `json:"transitions"`
}

// Run is `bundle/run.json`, the record of what was actually run. Everything
// time dependent is inside Variable and nothing time dependent is outside it,
// which is what makes a diff of two runs of the same case show physics.
type Run struct {
	// CaseID is the SHA-256 of the canonical case, in lowercase hexadecimal.
	CaseID string `json:"case-id"`
	// Participants is the participant identifiers, sorted ascending by code
	// point, which is the order every array keyed by participant uses.
	Participants []string `json:"participants"`
	// Engine and EngineVersion name the container engine that ran the codes.
	Engine        string `json:"engine"`
	EngineVersion string `json:"engine-version"`
	// Platform is the operating system and architecture the run happened on.
	Platform string `json:"platform"`
	// Harness is the version and commit of this tool.
	Harness WrittenBy `json:"harness"`
	// Constants is the CODATA revision the conversions were made under.
	// Changing it is a format bump, because it moves every converted number in
	// every bundle written afterwards.
	Constants string `json:"constants"`
	// Steps is one entry per participant, in Participants order.
	Steps []Step `json:"steps"`
	// Variable is the one fenced object 0007 permits to differ between two
	// runs of the same case against the same manifests.
	Variable Variable `json:"variable"`
}

// Step is what happened to one participant, apart from anything time dependent.
// Whether a limit fired and what exit status a step reported are here rather
// than in Variable on purpose: two runs differing in either are two different
// runs, and their bundles differ.
type Step struct {
	Participant string `json:"participant"`
	// Manifest is the digest of the container manifest the image carried,
	// which 0003 fixes the content of.
	Manifest string `json:"manifest"`
	// ExitStatus is the code's own exit status.
	ExitStatus int `json:"exit-status"`
	// LimitFired names the limit that stopped the step, or is empty where none
	// did.
	LimitFired string `json:"limit-fired"`
}

// Variable is the fenced object. Nothing outside it in any bundle file carries a
// clock reading, a duration, a measured resource figure, or an identifier
// derived from any of them.
type Variable struct {
	// RunID identifies this run within an operator's collection. It is the one
	// field here that is not a time reading, and it is here because it is
	// derived from one.
	RunID string `json:"run-id"`
	// Started and Finished are RFC 3339 timestamps in UTC.
	Started  string `json:"started"`
	Finished string `json:"finished"`
	// Steps carries the per participant timings, in Participants order.
	Steps []VariableStep `json:"steps"`
}

// VariableStep is one participant's timings and what it actually consumed.
// Both are needed to read a timeout: a step killed at its limit and a step that
// died at once are the two shapes a killed step takes, and a step that finished
// inside its wall clock limit while sitting against its memory limit is a
// result to read with suspicion.
type VariableStep struct {
	Participant string `json:"participant"`
	Started     string `json:"started"`
	Finished    string `json:"finished"`
	// PeakResidentBytes and CPUMilliseconds are what the step actually
	// consumed. Both are integers of a fixed smallest unit rather than
	// floating point figures, so that the only number in this model carried as
	// a float is a physical quantity with a unit and a significance count
	// beside it.
	PeakResidentBytes int `json:"peak-resident-bytes"`
	CPUMilliseconds   int `json:"cpu-milliseconds"`
}

// Manifest is `bundle/manifest.json`, what a third party consumer reads first.
// It is written so that reading it is enough to verify the bundle without 0007
// in hand.
type Manifest struct {
	// Format is the bundle format version. A consumer meeting a number it does
	// not know refuses the bundle and names both numbers rather than reading
	// on, which is what ReadManifest does.
	Format int `json:"bundle-format"`
	// CaseID is the SHA-256 of `bundle/case.json`, lowercase hexadecimal,
	// never truncated.
	CaseID string `json:"case-id"`
	// Participants is the participant identifiers, sorted ascending by code
	// point.
	Participants []string `json:"participants"`
	// WrittenBy names software and no person.
	WrittenBy WrittenBy `json:"written-by"`
	// Hash names the digest used for every checksum in the bundle. It is a
	// field rather than an assumption so that replacing it later is a format
	// bump and not an archaeology problem.
	Hash string `json:"hash"`
	// Members is every file under `bundle/` other than this one, sorted by
	// path. It is the authority for what the bundle contains: a file present
	// and unlisted invalidates the bundle, and so does the reverse.
	Members []Member `json:"members"`
	// Digest is the checksum over the members, defined over the listing rather
	// than over the file bytes so that a renamed file changes it.
	Digest string `json:"digest"`
	// Variable is the JSON pointers into `run.json` naming every field
	// permitted to differ between two runs. The list travels in the artefact
	// so a consumer diffing two bundles does not have to have read 0007.
	Variable []string `json:"variable"`
	// StableDigest is Digest computed with every field named in Variable
	// replaced by null. Two runs of the same case against the same manifests,
	// both completing the same way, have equal values here, and that equality
	// is the byte stability claim in a form a consumer can check with one
	// comparison.
	StableDigest string `json:"stable-digest"`
}

// WrittenBy is the version and commit of the software that wrote a bundle.
type WrittenBy struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// Member is one file under `bundle/` with its size and digest.
type Member struct {
	Path   string `json:"path"`
	Bytes  int    `json:"bytes"`
	Digest string `json:"digest"`
}
