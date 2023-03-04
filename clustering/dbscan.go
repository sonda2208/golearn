package clustering

import (
	"math/big"

	"gonum.org/v1/gonum/mat"

	"github.com/sjwhitworth/golearn/base"
	"github.com/sjwhitworth/golearn/metrics/pairwise"
)

// DBSCANParameters describes the parameters of the density-based
// clustering algorithm DBSCAN
type DBSCANParameters struct {
	ClusterParameters

	// Eps represents the "reachability", or the maximum
	// distance any point can be before being considered for
	// inclusion.
	Eps float64

	// MinCount represents how many points need to be
	// in a cluster before it is considered one.
	MinCount int
}

func regionQuery(p int, ret *big.Int, dist *mat.Dense, eps float64) *big.Int {
	rows, _ := dist.Dims()
	// Return any points within the Eps neighbourhood
	for i := 0; i < rows; i++ {
		if dist.At(p, i) <= eps {
			ret = ret.SetBit(ret, i, 1) // Mark as neighbour
		}
	}
	return ret
}

func computePairwiseDistances(inst base.FixedDataGrid, attrs []base.Attribute, metric pairwise.PairwiseDistanceFunc) (*mat.Dense, error) {
	// Compute pair-wise distances
	// First convert everything to floats
	mats, err := base.ConvertAllRowsToMat64(attrs, inst)
	if err != nil {
		return nil, err
	}

	// Next, do an n^2 computation of all pairwise distances
	_, rows := inst.Size()
	dist := mat.NewDense(rows, rows, nil)
	for i := 0; i < rows; i++ {
		for j := i + 1; j < rows; j++ {
			d := metric.Distance(mats[i], mats[j])
			dist.Set(i, j, d)
			dist.Set(j, i, d)
		}
	}
	return dist, nil
}

// DBSCAN clusters inst using the parameters allowed in and produces a ClusterId->[RowId] map
func DBSCAN(inst base.FixedDataGrid, params DBSCANParameters) (ClusterMap, error) {

	// Compute the distances between each possible point
	dist, err := computePairwiseDistances(inst, params.Attributes, params.Metric)
	if err != nil {
		return nil, err
	}

	_, rows := inst.Size()

	clusterMap := make(map[int][]int)
	visited := big.NewInt(0)
	clustered := big.NewInt(0)
	// expandCluster adds P to a cluster C, visiting any neighbours
	expandCluster := func(p int, neighbours *big.Int, c int) {
		if clustered.Bit(p) == 1 {
			return
		}
		// Add this point to cluster C
		if _, ok := clusterMap[c]; !ok {
			clusterMap[c] = make([]int, 0)
		}
		clusterMap[c] = append(clusterMap[c], p)
		clustered.SetBit(clustered, p, 1)
		visited.SetBit(visited, p, 1)

		for i := 0; i < rows; i++ {
			reset := false
			if neighbours.Bit(i) == 0 {
				// Not a neighbour, so skip
				continue
			}
			if visited.Bit(i) == 0 {
				// not yet visited
				visited = visited.SetBit(visited, i, 1) // Mark as visited
				newNeighbours := big.NewInt(0)
				newNeighbours = regionQuery(i, newNeighbours, dist, params.Eps)
				if BitCount(newNeighbours) >= params.MinCount {
					neighbours = neighbours.Or(neighbours, newNeighbours)
					reset = true
				}
			} else {
				continue
			}
			if clustered.Bit(i) == 0 {
				clusterMap[c] = append(clusterMap[c], i)
				clustered = clustered.SetBit(clustered, i, 1)
			}
			if reset {
				i = 0
			}
		}
	}

	c := 0
	for i := 0; i < rows; i++ {
		if visited.Bit(i) == 1 {
			continue // Already visited here
		}
		visited.SetBit(visited, i, 1)
		neighbours := big.NewInt(0)
		neighbours = regionQuery(i, neighbours, dist, params.Eps)
		if BitCount(neighbours) < params.MinCount {
			// Noise, cluster 0
			clustered = clustered.Or(clustered, neighbours)
			continue
		}
		c = c + 1 // Increment cluster count
		expandCluster(i, neighbours, c)
	}

	// Remove anything from the map which doesn't make
	// minimum points
	rmKeys := make([]int, 0)
	for id := range clusterMap {
		if len(clusterMap[id]) < params.MinCount {
			rmKeys = append(rmKeys, id)
		}
	}
	for _, r := range rmKeys {
		delete(clusterMap, r)
	}

	return ClusterMap(clusterMap), nil
}

// How many bits?
func BitCount(n *big.Int) int {
	var count int = 0
	for _, b := range n.Bytes() {
		count += int(bitCounts[b])
	}
	return count
}

// The bit counts for each byte value (0 - 255).
var bitCounts = []int8{
	// Generated by Java BitCount of all values from 0 to 255
	0, 1, 1, 2, 1, 2, 2, 3,
	1, 2, 2, 3, 2, 3, 3, 4,
	1, 2, 2, 3, 2, 3, 3, 4,
	2, 3, 3, 4, 3, 4, 4, 5,
	1, 2, 2, 3, 2, 3, 3, 4,
	2, 3, 3, 4, 3, 4, 4, 5,
	2, 3, 3, 4, 3, 4, 4, 5,
	3, 4, 4, 5, 4, 5, 5, 6,
	1, 2, 2, 3, 2, 3, 3, 4,
	2, 3, 3, 4, 3, 4, 4, 5,
	2, 3, 3, 4, 3, 4, 4, 5,
	3, 4, 4, 5, 4, 5, 5, 6,
	2, 3, 3, 4, 3, 4, 4, 5,
	3, 4, 4, 5, 4, 5, 5, 6,
	3, 4, 4, 5, 4, 5, 5, 6,
	4, 5, 5, 6, 5, 6, 6, 7,
	1, 2, 2, 3, 2, 3, 3, 4,
	2, 3, 3, 4, 3, 4, 4, 5,
	2, 3, 3, 4, 3, 4, 4, 5,
	3, 4, 4, 5, 4, 5, 5, 6,
	2, 3, 3, 4, 3, 4, 4, 5,
	3, 4, 4, 5, 4, 5, 5, 6,
	3, 4, 4, 5, 4, 5, 5, 6,
	4, 5, 5, 6, 5, 6, 6, 7,
	2, 3, 3, 4, 3, 4, 4, 5,
	3, 4, 4, 5, 4, 5, 5, 6,
	3, 4, 4, 5, 4, 5, 5, 6,
	4, 5, 5, 6, 5, 6, 6, 7,
	3, 4, 4, 5, 4, 5, 5, 6,
	4, 5, 5, 6, 5, 6, 6, 7,
	4, 5, 5, 6, 5, 6, 6, 7,
	5, 6, 6, 7, 6, 7, 7, 8,
}
