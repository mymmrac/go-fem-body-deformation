package main

import (
	"log/slog"
	"math"
	"slices"
	"time"

	"gonum.org/v1/exp/linsolve"
	"gonum.org/v1/gonum/mat"
)

var gaussian = [3]float64{-math.Sqrt(0.6), 0, math.Sqrt(0.6)}

var localPoints2D = [20][2]int{
	{-1, -1}, {1, -1}, {1, 1}, {-1, 1},
	{0, -1}, {1, 0}, {0, 1}, {-1, 0},
}

var localPoints3D = [20][3]int{
	{-1, 1, -1}, {1, 1, -1}, {1, -1, -1}, {-1, -1, -1},
	{-1, 1, 1}, {1, 1, 1}, {1, -1, 1}, {-1, -1, 1},
	{0, 1, -1}, {1, 0, -1}, {0, -1, -1}, {-1, 0, -1},
	{-1, 1, 0}, {1, 1, 0}, {1, -1, 0}, {-1, -1, 0},
	{0, 1, 1}, {1, 0, 1}, {0, -1, 1}, {-1, 0, 1},
}

var dfiabg [3 * 3 * 3][20][3]float64

func init() {
	calculateDFIABG()
}

func calculateDFIABG() {
	for k1, gamma := range gaussian {
		for k2, beta := range gaussian {
			for k3, alpha := range gaussian {
				for i, point := range localPoints3D {
					if i <= 7 {
						dfiabg[k1*9+k2*3+k3][i] = dfiabg18(alpha, beta, gamma, point[0], point[1], point[2])
					} else {
						dfiabg[k1*9+k2*3+k3][i] = dfiabg14(alpha, beta, gamma, point[0], point[1], point[2])
					}
				}
			}
		}
	}
}

func dfiabg18(alpha, beta, gamma float64, alphaI, betaI, gammaI int) [3]float64 {
	return [3]float64{
		(1.0 / 8.0) * (1 + beta*float64(betaI)) * (1 + gamma*float64(gammaI)) *
			(float64(alphaI)*(-2+alpha*float64(alphaI)+gamma*float64(gammaI)+beta*float64(betaI)) +
				float64(alphaI)*(1+alpha*float64(alphaI))),
		(1.0 / 8.0) * (1 + alpha*float64(alphaI)) * (1 + gamma*float64(gammaI)) *
			(float64(betaI)*(-2+alpha*float64(alphaI)+gamma*float64(gammaI)+beta*float64(betaI)) +
				float64(betaI)*(1+beta*float64(betaI))),
		(1.0 / 8.0) * (1 + beta*float64(betaI)) * (1 + alpha*float64(alphaI)) *
			(float64(gammaI)*(-2+alpha*float64(alphaI)+gamma*float64(gammaI)+beta*float64(betaI)) +
				float64(gammaI)*(1+gamma*float64(gammaI))),
	}
}

func dfiabg14(alpha, beta, gamma float64, alphaI, betaI, gammaI int) [3]float64 {
	return [3]float64{
		(1.0 / 4.0) * (1 + beta*float64(betaI)) * (1 + gamma*float64(gammaI)) *
			(float64(alphaI)*(-float64(betaI)*float64(betaI)*float64(gammaI)*float64(gammaI)*
				alpha*alpha-beta*beta*float64(gammaI)*float64(gammaI)*float64(alphaI)*float64(alphaI)-
				float64(betaI)*float64(betaI)*gamma*gamma*float64(alphaI)*float64(alphaI)+1) -
				(2*float64(betaI)*float64(betaI)*float64(gammaI)*float64(gammaI)*alpha)*(alpha*float64(alphaI)+1)),

		(1.0 / 4.0) * (1 + alpha*float64(alphaI)) * (1 + gamma*float64(gammaI)) *
			(float64(betaI)*(-float64(betaI)*float64(betaI)*float64(gammaI)*float64(gammaI)*
				alpha*alpha-beta*beta*float64(gammaI)*float64(gammaI)*float64(alphaI)*float64(alphaI)-
				float64(betaI)*float64(betaI)*gamma*gamma*float64(alphaI)*float64(alphaI)+1) -
				(2*beta*float64(gammaI)*float64(gammaI)*float64(alphaI)*float64(alphaI))*(float64(betaI)*beta+1)),

		(1.0 / 4.0) * (1 + beta*float64(betaI)) * (1 + alpha*float64(alphaI)) *
			(float64(gammaI)*(-float64(betaI)*float64(betaI)*float64(gammaI)*float64(gammaI)*
				alpha*alpha-beta*beta*float64(gammaI)*float64(gammaI)*float64(alphaI)*float64(alphaI)-
				float64(betaI)*float64(betaI)*gamma*gamma*float64(alphaI)*float64(alphaI)+1) -
				(2*float64(betaI)*float64(betaI)*gamma*float64(alphaI)*float64(alphaI))*(gamma*float64(gammaI)+1)),
	}
}

type FEM struct {
	elements [][20][3]float64 // Coords of grid vertices in local space, npq * 20 * 3 (x, y, z)
	akt      [][3]float64     // Coords of grid vertices in global space, npq * 3 (x, y, z)
	nt       [][20]int        // Local element indexes, npq * 20

	zu []int // Fixed points
	zp []int // Pushed points

	dj    [][27][3][3]float64 // Jacobian matrix, npq * 27 * 3 (a, b, g) * 3 (x, y, z)
	djDet [][27]float64       // Jacobian determinant, npq * 27

	dfixyz [][27][20][3]float64 // Derivative of approximation function in global space, npq * 27 * 20 * 3 (x, y, z)
}

func (f *FEM) Solve(bodySize [3]float64, bodySplit [3]int) {
	now := time.Now()
	defer func() { slog.Info("FEM", "time", time.Since(now)) }()

	slog.Info("FME", "DFIABG", dfiabg)

	f.fillElements(bodySize, bodySplit)
	slog.Info("FEM", "elements", f.elements)
	slog.Info("FEM", "AKT", f.akt)
	slog.Info("FEM", "NT", f.nt)

	f.dj = nil
	for _, cube := range f.elements {
		f.dj = append(f.dj, f.createDJ(cube))
	}
	slog.Info("FEM", "DJ", f.dj)

	f.djDet = nil
	for _, dj := range f.dj {
		var ds [27]float64
		for i, d := range dj {
			ds[i] = d[0][0]*d[1][1]*d[2][2] +
				d[0][1]*d[1][2]*d[2][0] +
				d[0][2]*d[1][0]*d[2][1] -
				d[0][2]*d[1][1]*d[2][0] - d[0][0]*
				d[1][2]*d[2][1] -
				d[0][1]*d[1][0]*d[2][2]
		}
		f.djDet = append(f.djDet, ds)
	}
	slog.Info("FEM", "DJDet", f.djDet)

	f.dfixyz = nil
	for _, dj := range f.dj {
		f.dfixyz = append(f.dfixyz, f.createDFIXYZ(dj))
	}
	slog.Info("FEM", "DFIXYZ", f.dfixyz)
}

func (f *FEM) fillElements(bodySize [3]float64, bodySplit [3]int) {
	stepA := bodySize[0] / float64(bodySplit[0])
	stepB := bodySize[1] / float64(bodySplit[1])
	stepC := bodySize[2] / float64(bodySplit[2])

	f.elements = nil
	for k := range bodySplit[2] {
		for j := range bodySplit[1] {
			for i := range bodySplit[0] {
				f.elements = append(f.elements, f.createCube(
					float64(i)*stepA, float64(i+1)*stepA,
					float64(j)*stepB, float64(j+1)*stepB,
					float64(k)*stepC, float64(k+1)*stepC,
				))
			}
		}
	}

	f.akt = nil
	for k := range 2*bodySplit[2] + 1 {
		if k%2 == 0 {
			for j := range 2*bodySplit[1] + 1 {
				if j%2 == 0 {
					for i := range 2*bodySplit[0] + 1 {
						f.akt = append(f.akt, [3]float64{float64(i) * stepA / 2, float64(j) * stepB / 2, float64(k) * stepC / 2})
					}
				} else {
					for i := range bodySplit[0] + 1 {
						f.akt = append(f.akt, [3]float64{float64(i) * stepA, float64(j) * stepB / 2, float64(k) * stepC / 2})
					}
				}
			}
		} else {
			for j := range bodySplit[1] + 1 {
				for i := range bodySplit[0] + 1 {
					f.akt = append(f.akt, [3]float64{float64(i) * stepA, float64(j) * stepB, float64(k) * stepC / 2})
				}
			}
		}
	}

	f.nt = nil
	for _, cube := range f.elements {
		var ntCube [20]int
		for i, xyz := range cube {
			ntCube[i] = slices.Index(f.akt, xyz)
		}
		f.nt = append(f.nt, ntCube)
	}
}

func (f *FEM) createCube(aStart, aEnd, bStart, bEnd, cStart, cEnd float64) [20][3]float64 {
	aSize := aEnd - aStart
	bSize := bEnd - bStart
	cSize := cEnd - cStart

	x := [20]float64{aStart, aEnd, aEnd, aStart, aStart, aEnd, aEnd, aStart,
		aStart + aSize/2, aEnd, aStart + aSize/2, aStart,
		aStart, aEnd, aEnd, aStart, aStart + aSize/2, aEnd,
		aStart + aSize/2, aStart}

	y := [20]float64{bStart, bStart, bEnd, bEnd, bStart, bStart, bEnd, bEnd,
		bStart, bStart + bSize/2, bEnd, bStart + bSize/2,
		bStart, bStart, bEnd, bEnd, bStart, bStart + bSize/2,
		bEnd, bStart + bSize/2}

	z := [20]float64{cStart, cStart, cStart, cStart, cEnd, cEnd, cEnd, cEnd,
		cStart, cStart, cStart, cStart, cStart + cSize/2,
		cStart + cSize/2, cStart + cSize/2, cStart + cSize/2,
		cEnd, cEnd, cEnd, cEnd}

	var cube [20][3]float64
	for i := range 20 {
		cube[i] = [3]float64{x[i], y[i], z[i]}
	}

	return cube
}

func (f *FEM) createDJ(cube [20][3]float64) [27][3][3]float64 {
	var dj [27][3][3]float64
	for i := range 27 {
		var sumXA, sumXB, sumXG float64
		var sumYA, sumYB, sumYG float64
		var sumZA, sumZB, sumZG float64

		for j, point := range cube {
			sumXA += point[0] * dfiabg[i][j][0]
			sumXB += point[0] * dfiabg[i][j][1]
			sumXG += point[0] * dfiabg[i][j][2]

			sumYA += point[1] * dfiabg[i][j][0]
			sumYB += point[1] * dfiabg[i][j][1]
			sumYG += point[1] * dfiabg[i][j][2]

			sumZA += point[2] * dfiabg[i][j][0]
			sumZB += point[2] * dfiabg[i][j][1]
			sumZG += point[2] * dfiabg[i][j][2]
		}

		dj[i] = [3][3]float64{
			{sumXA, sumYA, sumZA},
			{sumXB, sumYB, sumZB},
			{sumXG, sumYG, sumZG},
		}
	}
	return dj
}

func (f *FEM) createDFIXYZ(dj [27][3][3]float64) [27][20][3]float64 {
	var dfixyz [27][20][3]float64
	for i, d := range dj {
		for j, points := range dfiabg[i] {
			a := mat.NewDense(3, 3, []float64{
				d[0][0], d[0][1], d[0][2],
				d[1][0], d[1][1], d[1][2],
				d[2][0], d[2][1], d[2][2],
			})
			b := mat.NewVecDense(3, points[:])

			result, err := linsolve.Iterative(&matrix{Dense: a}, b, &linsolve.CG{}, nil)
			if err != nil {
				dfixyz[i][j] = [3]float64{0, 0, 0}
			} else {
				dfixyz[i][j] = [3]float64{result.X.AtVec(0), result.X.AtVec(1), result.X.AtVec(2)}
			}
		}
	}
	return dfixyz
}

type matrix struct {
	*mat.Dense
}

func (m *matrix) MulVecTo(dst *mat.VecDense, trans bool, x mat.Vector) {
	if trans {
		dst.MulVec(m.T(), x)
	} else {
		dst.MulVec(m.Dense, x)
	}
}
